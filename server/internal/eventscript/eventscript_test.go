package eventscript

import "testing"

func TestToLisp(t *testing.T) {
	cases := map[string]string{
		"start_quest(1)":                          "(start-quest 1)",
		"finish_quest(2)":                         "(finish-quest 2)",
		"check_point(1, 14, 3)":                   "(check-point 1 14 3)",
		"open_teleport(1)":                        "(open-teleport 1)",
		"open_bank":                               "(open-bank)",
		"@client.add_vip_days(5)":                 "(add-vip-days 5)",
		"chat_add('hello', Configs::ERROR_COLOR)": `(chat-add "hello" 10)`,
		"chat_add('a\nb', 5)\nstart_quest(3)":     `(begin (chat-add "a b" 5) (start-quest 3))`,
	}
	for source, want := range cases {
		got, ok := ToLisp(source)
		if !ok {
			t.Fatalf("ToLisp(%q) failed", source)
		}
		if got != want {
			t.Errorf("ToLisp(%q) = %q, want %q", source, got, want)
		}
	}
	if _, ok := ToLisp("[NG] some_client_code"); ok {
		t.Fatal("[NG] script translated")
	}
	if _, ok := ToLisp("puts 'unknown'"); ok {
		t.Fatal("unknown script translated")
	}
}
