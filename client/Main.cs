using Godot;
using System;
using Microsoft.Scripting.Hosting;

public class Main : Node2D
{
    ScriptEngine _engine;
    ScriptRuntime _runtime;
    ScriptScope _scope;
    dynamic _main;
    bool _hasError;
    AcceptDialog _errorDialog;

    public override void _Ready()
    {
        SetupErrorDialog();
        SetupRubyEnvironment();
        RunScripts();
        RaiseException("error message here");
    }

    public override void _Process(float delta)
    {
        _engine.Operations.InvokeMember(_main, "process", delta);
    }

    void SetupErrorDialog()
    {
        _errorDialog = GetNode<AcceptDialog>("ErrorDialog");
        _errorDialog.WindowTitle = "Error";
        _errorDialog.Connect("confirmed", this, nameof(OnDialogConfirmed));
    }

    void OnDialogConfirmed() => GetTree().Quit();

    void RaiseException(string message)
    {
        _hasError = true;
        _errorDialog.DialogText = message;
        _errorDialog.PopupCentered();
    }

    void SetupRubyEnvironment()
    {
        _runtime = IronRuby.Ruby.CreateRuntime();
        _engine = _runtime.GetEngine("Ruby");
        _scope = _engine.CreateScope();
    }

    string LoadScripts()
    {
        var file = new File();
        file.Open("res://main.rb", File.ModeFlags.Read);
        var content = file.GetAsText();
        file.Close();
        return content;
    }

    void RunScripts()
    {
        _engine.Execute(LoadScripts(), _scope);
        var mainClass = _engine.Runtime.Globals.GetVariable("Main");
        _main = _engine.Operations.CreateInstance(mainClass);
    }
}
