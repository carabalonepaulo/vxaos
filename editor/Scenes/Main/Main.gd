tool
extends Control


export(NodePath) var map_editor_path
export(NodePath) var database_editor_path
export(NodePath) var scripts_editor_path
export(NodePath) var settings_path

onready var main_views: Dictionary = {
    Constants.MenuAction.OpenMapEditor: map_editor_path,
    Constants.MenuAction.OpenDatabaseEditor: database_editor_path,
    Constants.MenuAction.OpenScriptsEditor: scripts_editor_path,
    Constants.MenuAction.OpenSettings: settings_path
}

func _ready():
    _on_menu_action(Constants.MenuAction.OpenMapEditor)

func _on_menu_action(id):
    if not main_views.has(id): return
    for key in main_views:
        (get_node(main_views[key]) as Control).visible = false
    (get_node(main_views[id]) as Control).visible = true
