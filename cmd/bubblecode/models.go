package bubblecode

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
)

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Manage configured models",
	}
	cmd.AddCommand(modelsListCmd())
	cmd.AddCommand(modelsSwitchCmd())
	cmd.AddCommand(modelsConfigCmd())
	return cmd
}

func modelsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured models",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agent.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := agent.LoadConfig(path)
			if err != nil {
				fmt.Println("No config found. Use 'bubblecode providers config' to set up.")
				return nil
			}
			fmt.Printf("Default: %s / %s\n\n", cfg.DefaultProvider, cfg.DefaultModel)
			var n int
			for _, p := range cfg.Providers {
				for _, m := range p.Models {
					n++
					dm := m.ID
					if p.Name == cfg.DefaultProvider && m.ID == cfg.DefaultModel {
						dm += " [default]"
					}
					fmt.Printf("%2d. %s/%s\n", n, p.Name, dm)
				}
			}
			if n == 0 {
				fmt.Println("No models configured. Use 'bubblecode providers add' or 'bubblecode providers config'.")
			}
			return nil
		},
	}
}

func modelsSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch [model]",
		Short: "Switch active model for current provider",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadProviders()
			if err != nil {
				return err
			}
			p := cfg.GetDefaultProvider()
			if p == nil {
				return fmt.Errorf("no active provider")
			}
			model := ""
			if len(args) > 0 {
				model = args[0]
				if idx, err := strconv.Atoi(model); err == nil && idx >= 1 && idx <= len(p.Models) {
					model = p.Models[idx-1].ID
				}
			} else if len(p.Models) > 0 {
				opts := make([]huh.Option[string], len(p.Models))
				for i, m := range p.Models {
					opts[i] = huh.NewOption(m.ID, m.ID)
				}
				err := huh.NewSelect[string]().
					Title("Select model").
					Options(opts...).
					Value(&model).
					Run()
				if err != nil {
					return err
				}
			}
			if model == "" {
				return fmt.Errorf("no model specified and none available")
			}
			cfg.DefaultModel = model
			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Model switched to: %s\n", model)
			return nil
		},
	}
}

func modelsConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Interactive model configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := agent.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := agent.LoadConfig(path)
			if err != nil {
				fmt.Println("No config found. Use 'bubblecode providers config' to set up.")
				return nil
			}

			for {
				fmt.Printf("\n  Default: %s / %s\n\n", cfg.DefaultProvider, cfg.DefaultModel)
				var n int
				for _, p := range cfg.Providers {
					for _, m := range p.Models {
						n++
						dm := m.ID
						if p.Name == cfg.DefaultProvider && m.ID == cfg.DefaultModel {
							dm += " [default]"
						}
						fmt.Printf("  %2d. %s/%s\n", n, p.Name, dm)
					}
				}
				fmt.Println()

				var choice string
				opts := []huh.Option[string]{
					huh.NewOption("Switch default model", "switch"),
					huh.NewOption("Done", "quit"),
				}
				err := huh.NewSelect[string]().
					Title("Model Config").
					Options(opts...).
					Value(&choice).
					Run()
				if err != nil {
					return err
				}

				switch choice {
				case "quit":
					if err := agent.SaveConfig(path, cfg); err != nil {
						return fmt.Errorf("save config: %w", err)
					}
					fmt.Println("Config saved.")
					return nil

				case "switch":
					p := cfg.GetDefaultProvider()
					if p == nil || len(p.Models) == 0 {
						fmt.Println("No models available.")
						continue
					}
					modelOpts := make([]huh.Option[string], len(p.Models))
					for i, m := range p.Models {
						label := m.ID
						if m.ID == cfg.DefaultModel {
							label += " [default]"
						}
						modelOpts[i] = huh.NewOption(label, m.ID)
					}
					var model string
					huh.NewSelect[string]().
						Title("Select default model").
						Options(modelOpts...).
						Value(&model).
						Run()
					if model != "" {
						cfg.DefaultModel = model
					}
				}
			}
		},
	}
}
