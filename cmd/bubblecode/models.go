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
			fmt.Printf("Active: %s / %s\n\n", cfg.ActiveProvider, cfg.ActiveModel)
			var n int
			for _, p := range cfg.Providers {
				for _, m := range p.Models {
					n++
					mm := m
					if p.Name == cfg.ActiveProvider && m == cfg.ActiveModel {
						mm += " [default]"
					}
					fmt.Printf("%2d. %s/%s\n", n, p.Name, mm)
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
			p := cfg.GetActiveProvider()
			if p == nil {
				return fmt.Errorf("no active provider")
			}
			model := ""
			if len(args) > 0 {
				model = args[0]
				if idx, err := strconv.Atoi(model); err == nil && idx >= 1 && idx <= len(p.Models) {
					model = p.Models[idx-1]
				}
			} else if len(p.Models) > 0 {
				opts := make([]huh.Option[string], len(p.Models))
				for i, m := range p.Models {
					opts[i] = huh.NewOption(m, m)
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
			cfg.ActiveModel = model
			if err := agent.SaveConfig(path, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("Model switched to: %s\n", model)
			return nil
		},
	}
}
