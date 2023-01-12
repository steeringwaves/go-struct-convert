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

var inputFiles []string
var dirname string = ""
var prefix string = ""
var suffix string = ""
var name string = ""
var cIncludes []string
var tsNamespace string = ""
var tsImports []string

// var tsRequires []string
var useStdout bool = true

func doConversion(input []string, output string, c converter.Converter) {
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

	builder, err := converter.ConvertFile(input, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if useStdout {
		fmt.Println(builder.String())
	}

	err = ioutil.WriteFile(path.Join(dirname, fmt.Sprintf("%s.%s", output, c.FileExtension())), []byte(builder.String()), 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var typescriptCmd = &cobra.Command{
	Use:   "typescript",
	Short: "Converts go structs to typescript",
	Long:  `This command converts go structs to typescript`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(inputFiles) == 0 {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "no files specified")
				os.Exit(1)
			}

			for i := range args {
				inputFiles = append(inputFiles, args[i])
			}
		}

		outputFilename := name
		if outputFilename == "" {
			outputFilename = strings.TrimSuffix(path.Base(inputFiles[0]), path.Ext(inputFiles[0]))
		} else {
			outputFilename = strings.TrimSuffix(path.Base(outputFilename), path.Ext(outputFilename))
		}

		doConversion(inputFiles, outputFilename, &converter.TypescriptConverter{
			Prefix:    prefix,
			Suffix:    suffix,
			Namespace: tsNamespace,
			Comments: converter.Comments{
				TypescriptImports: tsImports,
				// TypescriptRequires: tsRequires,
			},
		})
	},
}

var cCmd = &cobra.Command{
	Use:   "c",
	Short: "Converts go structs to c",
	Long:  `This command converts go structs to c`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(inputFiles) == 0 {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "no files specified")
				os.Exit(1)
			}

			for i := range args {
				inputFiles = append(inputFiles, args[i])
			}
		}

		outputFilename := name
		if outputFilename == "" {
			outputFilename = strings.TrimSuffix(path.Base(inputFiles[0]), path.Ext(inputFiles[0]))
		} else {
			outputFilename = strings.TrimSuffix(path.Base(outputFilename), path.Ext(outputFilename))
		}

		doConversion(inputFiles, outputFilename, &converter.CConverter{
			Prefix: prefix,
			Suffix: suffix,
			Comments: converter.Comments{
				CIncludes: cIncludes,
			},
		})
	},
}

func main() {
	var rootCmd = &cobra.Command{Use: os.Args[0]}

	rootCmd.PersistentFlags().StringSliceVarP(&inputFiles, "input", "i", []string{}, "the input file(s) to parse")
	rootCmd.PersistentFlags().StringVarP(&dirname, "output", "o", "", "the output directory to save to instead of stdout")
	rootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "the name for the output file (extension is added automatically)")
	rootCmd.PersistentFlags().StringVarP(&prefix, "prefix", "", "", "the prefix for each struct name to add")
	rootCmd.PersistentFlags().StringVarP(&suffix, "suffix", "", "", "the suffix for each struct name to add")

	// TODO cobra won't let users specify --include "#include \"myfile.h\"" (it doesn't like the quotes)
	cCmd.Flags().StringSliceVarP(&cIncludes, "include", "", []string{}, "include statements to add (do not include #include it will be added automatically)")

	typescriptCmd.Flags().StringVarP(&tsNamespace, "namespace", "", "", "the namespace to create and nest all interfaces under")
	typescriptCmd.Flags().StringSliceVarP(&tsImports, "import", "", []string{}, "import statements to add")

	// TODO if we need to add require statements, it will be messy dealing with cleaning the string `const { mything, anotherthing } = require('lodash');`
	// typescriptCmd.Flags().StringSliceVarP(&tsRequires, "require", "", []string{}, "require statements to add")

	rootCmd.AddCommand(typescriptCmd)
	rootCmd.AddCommand(cCmd)

	rootCmd.Execute()
}
