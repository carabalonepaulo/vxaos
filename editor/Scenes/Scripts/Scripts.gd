extends Control


export(NodePath) var text_edit_path

func _ready():
    var text_edit: TextEdit = get_node(text_edit_path)
    text_edit.add_keyword_color("class", Color.cornflower)
    text_edit.add_keyword_color("end", Color.cornflower)
    text_edit.add_keyword_color("def", Color.cornflower)
#    text_edit.
