package cmd

import (
	"bgate/model"
	"bgate/search"
	"bgate/view"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

var root = &cobra.Command{
	Use:   "bgate [flags] <query>",
	Short: "A terminal interface to Bible Gateway",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("translation", cmd.Flag("translation"))
		viper.BindPFlag("interactive", cmd.Flag("interactive"))
		viper.BindPFlag("padding", cmd.Flag("padding"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		translation := viper.GetString("translation")
		query := args[0]
		interactive := viper.GetBool("interactive")
		padding := viper.GetInt("padding")

		document, err := search.Passage(translation, query)
		cobra.CheckErr(err)

		document.Find(".crossreference").Remove()
		document.Find(".footnote").Remove()

		content := []model.Content{}
		document.Find(".passage-content").Each(func(pi int, passage *goquery.Selection) {
			passage.Find(".text").Each(func(li int, line *goquery.Selection) {
				if strings.HasPrefix(line.Parent().Nodes[0].Data, "h") {
					content = append(content, model.Content{
						Type:    model.Section,
						Content: line.Text(),
					})
					return
				}

				chapter := line.Find(".chapternum")
				if chapter.Length() > 0 {
					c := model.Content{
						Type:   model.Chapter,
						Number: chapter.Text(),
					}
					chapter.Remove()
					content = append(content, c)

					c.Type = model.Verse
					c.Number = "1 "
					c.Content = line.Text()
					content = append(content, c)
					return
				}

				verse := line.Find(".versenum")
				if verse.Length() > 0 {
					c := model.Content{
						Type:   model.Verse,
						Number: verse.Text(),
					}
					verse.Remove()

					c.Content = line.Text()
					content = append(content, c)
					return
				}

				content = append(content, model.Content{
					Type:    model.VerseCont,
					Content: line.Text(),
				})
			})
		})

		if len(content) == 0 {
			cobra.CheckErr(errors.New("No content found"))
		}

		m := view.New(content, padding)
		if !interactive {
			width, _, err := terminal.GetSize(0)
			if err != nil {
				panic(err)
			}
			m.SetWindowSize(width, math.MaxInt32)
			v := m.View()
			fmt.Print(v)
			return
		}

		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	var config string
	root.Flags().StringVarP(&config, "config", "c", "~/.config/bgate/config.json", "Config file to use.")
	root.Flags().StringP("translation", "t", "", "The translation of the Bible to search for.")
	root.Flags().BoolP("interactive", "i", false, "Interactive view, allows you to scroll using j/up and k/down.")
	root.Flags().IntP("padding", "p", 0, "Horizontal padding in character count.")

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	config = strings.ReplaceAll(config, "~", home)
	viper.SetConfigFile(config)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}
}
