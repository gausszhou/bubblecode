package bubblecode

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gausszhou/bubblecode/agent"
)

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Show or switch configured models",
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
			var n int
			for _, p := range cfg.Providers {
				for _, m := range p.Models {
					n++
					mm := m
					if p.Name == cfg.ActiveProvider && m == cfg.ActiveModel {
						mm += " *"
					}
					fmt.Printf("%2d. %s/%s\n", n, p.Name, mm)
				}
			}
			return nil
		},
	}
	cmd.AddCommand(modelsSwitchCmd())
	return cmd
}

func modelsSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <model>",
		Short: "Switch active model for current provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadProviders()
			if err != nil {
				return err
			}
			p := cfg.GetActiveProvider()
			if p == nil {
				return fmt.Errorf("no active provider")
			}
			model := args[0]
			if idx, err := strconv.Atoi(model); err == nil && idx >= 1 && idx <= len(p.Models) {
				model = p.Models[idx-1]
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
