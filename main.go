// wllinear is a Wayland-native GUI client for Linear.app.
package main

import (
	"fmt"
	"log"
	"os"

	gioapp "gioui.org/app"
	"gioui.org/unit"

	wllapp "github.com/denislee/wllinear/internal/app"
	"github.com/denislee/wllinear/internal/config"
	"github.com/denislee/wllinear/internal/db"
	"github.com/denislee/wllinear/internal/linear"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	saved := config.LoadState()
	client := linear.NewClient(cfg.APIKey)

	d, err := db.Open(config.DBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
	}
	defer db.Close(d)

	go func() {
		w := new(gioapp.Window)
		w.Option(
			gioapp.Title("wllinear"),
			gioapp.Size(unit.Dp(1200), unit.Dp(800)),
			gioapp.MinSize(unit.Dp(720), unit.Dp(480)),
		)

		a := wllapp.NewApp(w, client, d, cfg, saved)
		if err := a.Run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	gioapp.Main()
}
