package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/steeringwaves/go-struct-convert/converter"
)

var dirname string = ""

var typescriptCmd = &cobra.Command{
	Use:   "typescript",
	Short: "Converts go structs to typescript",
	Long:  `This command converts go structs to typescript`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "no files specified")
			os.Exit(1)
		}

		for _, filename := range args {
			builder, err := converter.ConvertFile(filename, &converter.TypescriptConverter{})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println(builder.String())
		}
	},
}

var cCmd = &cobra.Command{
	Use:   "c",
	Short: "Converts go structs to c",
	Long:  `This command converts go structs to c`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "no files specified")
			os.Exit(1)
		}

		for _, filename := range args {
			builder, err := converter.ConvertFile(filename, &converter.CConverter{})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println(builder.String())
		}
	},
}

func main() {
	var rootCmd = &cobra.Command{Use: os.Args[0]}

	rootCmd.PersistentFlags().StringVarP(&dirname, "output", "o", "", "the output directory to save to instead of stdout")

	rootCmd.AddCommand(typescriptCmd)
	rootCmd.AddCommand(cCmd)

	rootCmd.Execute()
}
