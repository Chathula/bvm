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

// Install downloads a Bun version ("latest" allowed) and activates it.
func Install(arg string) error {
	platform, err := util.DetectPlatform()
	if err != nil {
		return fail("Platform not supported: %v", err)
	}

	version, err := util.ResolveTarget(arg)
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
	if err := download(platform.DownloadURL(version), archivePath); err != nil {
		os.RemoveAll(dir)
		return fail("Failed downloading bun %s: %v", version, err)
	}

	if _, err := util.ExtractBunBinary(archivePath, dir); err != nil {
		os.RemoveAll(dir)
		return fail("%v", err)
	}

	if err := os.Remove(archivePath); err != nil {
		return fail("%v", err)
	}

	if err := util.Activate(version); err != nil {
		return fail("Installed but failed to activate: %v", err)
	}

	changed, err := util.EnsurePATH()
	if err != nil {
		fmt.Println(color.YellowString("Warning: could not update shell profile: %v", err))
	} else if changed {
		fmt.Println(color.YellowString("Added %s to your PATH — restart your shell or source your profile.", "~/.bun/bin"))
	}

	fmt.Println(color.GreenString("Successfully installed and activated bun %s", version))
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
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	closeErr := f.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dest)
}
