package bubblecode

import (
	"fmt"

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
			model := args[0]
			p := cfg.GetActiveProvider()
			if p == nil {
				return fmt.Errorf("no active provider")
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
