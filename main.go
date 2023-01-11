package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/steeringwaves/go-struct-convert/converter"
)

var dirname string = ""
var prefix string = ""
var suffix string = ""
var outputFilename string = ""
var useStdout bool = true

func doConversion(args []string, c converter.Converter) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "no files specified")
		os.Exit(1)
	}

	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "too many files specified")
		os.Exit(1)
	}

	if dirname != "" {
		useStdout = false
		dirname = path.Clean(dirname)

		stat, err := os.Stat(dirname)
		if err == nil {
			if !stat.IsDir() {
				fmt.Fprintln(os.Stderr, dirname, "already exists but is not a directory")
				os.Exit(1)
			}
		} else {
			err := os.MkdirAll(dirname, 0755)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}

	for _, filename := range args {
		builder, err := converter.ConvertFile(filename, c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if useStdout {
			fmt.Println(builder.String())
			continue
		}

		out := outputFilename
		if out == "" {
			out = strings.TrimSuffix(path.Base(filename), path.Ext(filename))
		} else {
			out = strings.TrimSuffix(path.Base(out), path.Ext(out))
		}

		err = ioutil.WriteFile(path.Join(dirname, fmt.Sprintf("%s.%s", out, c.FileExtension())), []byte(builder.String()), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

var typescriptCmd = &cobra.Command{
	Use:   "typescript",
	Short: "Converts go structs to typescript",
	Long:  `This command converts go structs to typescript`,
	Run: func(cmd *cobra.Command, args []string) {
		doConversion(args, &converter.TypescriptConverter{
			Prefix: prefix,
			Suffix: suffix,
		})
	},
}

var cCmd = &cobra.Command{
	Use:   "c",
	Short: "Converts go structs to c",
	Long:  `This command converts go structs to c`,
	Run: func(cmd *cobra.Command, args []string) {
		doConversion(args, &converter.CConverter{
			Prefix: prefix,
			Suffix: suffix,
		})
	},
}

func main() {
	var rootCmd = &cobra.Command{Use: os.Args[0]}

	rootCmd.PersistentFlags().StringVarP(&dirname, "output", "o", "", "the output directory to save to instead of stdout")
	rootCmd.PersistentFlags().StringVarP(&outputFilename, "name", "n", "", "the name for the output file (extension is added automatically)")
	rootCmd.PersistentFlags().StringVarP(&prefix, "prefix", "", "", "the prefix for each struct name to add")
	rootCmd.PersistentFlags().StringVarP(&suffix, "suffix", "", "", "the suffix for each struct name to add")

	rootCmd.AddCommand(typescriptCmd)
	rootCmd.AddCommand(cCmd)

	rootCmd.Execute()
}
