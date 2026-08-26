package command

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Installable seams so the full install flow is unit-testable offline.
var (
	detectPlatformFn = util.DetectPlatform
	downloadFile     = download
	ensurePATH       = util.EnsurePATH
)

// Install downloads a Bun version and activates it. The version may be given
// explicitly ("latest" allowed) or via a .bvmrc file in the current directory.
func Install(arg string) error {
	arg = resolveVersionArg(arg)
	if arg == "" {
		return fail("no version given and no %s found in the current directory", rcFileName)
	}

	platform, err := detectPlatformFn()
	if err != nil {
		return fail("Platform not supported: %v", err)
	}

	version, err := resolveTarget(arg)
	if err != nil {
		return fail("%v", err)
	}

	if util.IsInstalled(version) {
		fmt.Println(color.YellowString("bun %s is already installed. Run 'bvm use %s' to activate it.", version, version))
		return nil
	}

	dir, err := util.VersionDir(version)
	if err != nil {
		return fail("%v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail("%v", err)
	}

	archivePath := filepath.Join(dir, platform.AssetName())
	if err := downloadFile(platform.DownloadURL(version), archivePath); err != nil {
		os.RemoveAll(dir)
		return fail("Failed downloading bun %s: %v", version, err)
	}
	defer os.Remove(archivePath) // failure paths drop the whole dir anyway

	if _, err := util.ExtractBunBinary(archivePath, dir); err != nil {
		os.RemoveAll(dir)
		return fail("%v", err)
	}

	if err := util.Activate(version); err != nil {
		return fail("Installed but failed to activate: %v", err)
	}

	// First ever install records itself as the default alias.
	if err := ensureDefaultSet(version); err != nil {
		fmt.Println(color.YellowString("Warning: could not set default alias: %v", err))
	}

	changed, err := ensurePATH()
	if err != nil {
		fmt.Println(color.YellowString("Warning: could not update shell profile: %v", err))
	} else if changed {
		fmt.Println(color.YellowString("Added %s to your PATH — restart your shell or source your profile.", "~/.bun/bin"))
	}

	fmt.Println(color.GreenString("Successfully installed and activated bun %s", version))
	return nil
}

// ensureDefaultSet records version as the default alias, but only when no
// default exists yet — the first install wins, later installs never steal it.
var ensureDefaultSet = func(version string) error {
	if def, _ := util.DefaultVersion(); def != "" {
		return nil
	}
	if err := util.SetDefaultVersion(version); err != nil {
		return err
	}
	fmt.Println(color.HiBlackString("Set as default bun version"))
	return nil
}

func download(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("release archive not found (%s)", url)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(resp.ContentLength, "downloading")
	_, copyErr := io.Copy(io.MultiWriter(f, bar), resp.Body)
	closeErr := f.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	return os.Rename(tmp, dest)
}
