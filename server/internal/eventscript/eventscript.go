// Package eventscript translates the Ruby helper calls used by event command
// 355 into the Lisp executed by the server. Unsupported scripts remain on the
// client until they are ported.
package eventscript

import (
	"fmt"
	"regexp"
	"strings"
)

type statement struct {
	re   *regexp.Regexp
	form string
}

var statements = []statement{
	{regexp.MustCompile(`^start_quest\s*\(\s*(-?\d+)\s*\)`), "(start-quest %s)"},
	{regexp.MustCompile(`^finish_quest\s*\(\s*(-?\d+)\s*\)`), "(finish-quest %s)"},
	{regexp.MustCompile(`^check_point\s*\(\s*(-?\d+)\s*,\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`), "(check-point %s %s %s)"},
	{regexp.MustCompile(`^open_teleport\s*\(\s*(-?\d+)\s*\)`), "(open-teleport %s)"},
	{regexp.MustCompile(`^open_bank\s*\(?\s*\)?`), "(open-bank)"},
	{regexp.MustCompile(`^open_create_guild\s*\(?\s*\)?`), "(open-create-guild)"},
	{regexp.MustCompile(`^open_shop\s*\(?\s*\)?`), "(open-shop)"},
	{regexp.MustCompile(`^@client\.add_vip_days\s*\(\s*(-?\d+)\s*\)`), "(add-vip-days %s)"},
	{regexp.MustCompile(`^@client\.reset_parameters\s*\(?\s*\)?`), "(reset-parameters)"},
	{regexp.MustCompile(`^@client\.change_sex\s*\(?\s*\)?`), "(change-sex)"},
	{regexp.MustCompile(`^@client\.save_original_graphic\s*\(?\s*\)?`), "(save-original-graphic)"},
	{regexp.MustCompile(`^@client\.load_original_graphic\s*\(?\s*\)?`), "(load-original-graphic)"},
	{regexp.MustCompile(`^\$network\.send_player_graphic\s*\(\s*@client\s*\)`), "(send-player-graphic)"},
	{regexp.MustCompile(`^\$network\.send_player_sex\s*\(\s*@client\s*\)`), "(send-player-sex)"},
	{regexp.MustCompile(`^@client\.erase_buff\s*\(\s*(-?\d+)\s*\)`), "(erase-buff %s)"},
}

var chatSingleRE = regexp.MustCompile(`(?s)^chat_add\s*\(\s*'((?:\\.|[^'])*)'\s*(?:,\s*([A-Za-z0-9_:]+)\s*)?\)`)
var chatDoubleRE = regexp.MustCompile(`(?s)^chat_add\s*\(\s*"((?:\\.|[^"])*)"\s*(?:,\s*([A-Za-z0-9_:]+)\s*)?\)`)

// ToLisp converts a supported script to one Lisp expression. It returns false
// for anything it does not understand, including [NG] client-only scripts.
func ToLisp(script string) (string, bool) {
	text := strings.TrimSpace(script)
	if text == "" || strings.HasPrefix(text, "[NG]") {
		return "", false
	}
	if strings.HasPrefix(text, "(") {
		return text, true
	}
	var forms []string
	for {
		text = strings.TrimLeft(text, " \t\r\n;")
		if text == "" {
			break
		}
		form, rest, ok := one(text)
		if !ok {
			return "", false
		}
		forms = append(forms, form)
		text = rest
	}
	if len(forms) == 0 {
		return "", false
	}
	if len(forms) == 1 {
		return forms[0], true
	}
	return "(begin " + strings.Join(forms, " ") + ")", true
}

func one(text string) (string, string, bool) {
	for _, re := range []*regexp.Regexp{chatSingleRE, chatDoubleRE} {
		match := re.FindStringSubmatchIndex(text)
		if match == nil || match[0] != 0 {
			continue
		}
		body := text[match[2]:match[3]]
		color := "1"
		if match[4] >= 0 {
			color = text[match[4]:match[5]]
		}
		switch color {
		case "Configs::ERROR_COLOR":
			color = "10"
		case "Configs::SUCCESS_COLOR":
			color = "5"
		case "Configs::GLOBAL_COLOR":
			color = "1"
		}
		return fmt.Sprintf("(chat-add %q %s)", cleanRubyString(body), color), text[match[1]:], true
	}
	for _, st := range statements {
		match := st.re.FindStringSubmatchIndex(text)
		if match == nil || match[0] != 0 {
			continue
		}
		args := make([]any, 0, (len(match)-2)/2)
		for i := 2; i < len(match); i += 2 {
			if match[i] < 0 {
				continue
			}
			args = append(args, text[match[i]:match[i+1]])
		}
		form := st.form
		if len(args) > 0 {
			form = fmt.Sprintf(st.form, args...)
		}
		return form, text[match[1]:], true
	}
	return "", text, false
}

func cleanRubyString(value string) string {
	value = strings.ReplaceAll(value, `\'`, `'`)
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
