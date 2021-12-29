extends HBoxContainer


signal action(id)


func _on_project_pressed():
    emit_signal("action", Constants.MenuAction.OpenProject)

func _on_map_pressed():
    emit_signal("action", Constants.MenuAction.OpenMapEditor)

func _on_database_pressed():
    emit_signal("action", Constants.MenuAction.OpenDatabaseEditor)

func _on_scripts_pressed():
    emit_signal("action", Constants.MenuAction.OpenScriptsEditor)

func _on_settings_pressed():
    emit_signal("action", Constants.MenuAction.OpenSettings)
