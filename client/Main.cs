using Godot;
using System;
using Microsoft.Scripting.Hosting;
using System.Collections.Generic;

public interface RubyAPI
{
    event Action<String> ExceptionRaised;
}

public class Main : Node2D
{
    ScriptEngine _engine;
    ScriptRuntime _runtime;
    ScriptScope _scope;
    dynamic _main;
    bool _hasError;
    AcceptDialog _errorDialog;

    API.System _systemAPI;
    API.Primitives _primitivesAPI;

    public static ScriptEngine RubyEngine;

    public override void _Ready()
    {
        SetupErrorDialog();
        CreateAPIs();
        SetupRubyEnvironment();
        LoadScripts();
        Run();
    }

    public override void _Process(float delta)
    {
        if (_hasError || _main == null) return;
        _engine.Operations.InvokeMember(_main, "process", delta);
    }

    void SetupErrorDialog()
    {
        _errorDialog = GetNode<AcceptDialog>("ErrorDialog");
        _errorDialog.WindowTitle = "Error";
        _errorDialog.Connect("confirmed", this, nameof(OnDialogConfirmed));
    }

    void OnDialogConfirmed() => GetTree().Quit();

    void CreateAPIs()
    {
        _systemAPI = new API.System(this);
        _systemAPI.ExceptionRaised += RaiseException;
        _primitivesAPI = new API.Primitives();
    }

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
        RubyEngine = _engine;
    }

    string[] CollectScriptFiles(string dirPath)
    {
        var dir = new Directory();
        dir.Open(dirPath);
        dir.ListDirBegin();

        var files = new List<string>();
        var fileName = dir.GetNext();
        while (fileName != "")
        {
            if (fileName.Find(".rb") != -1 && fileName != "main.rb")
                files.Add(fileName);
            fileName = dir.GetNext();
        }

        var file = new File();
        if (file.FileExists($"{dirPath}main.rb")) files.Add("main.rb");
        else RaiseException("Main script not found.");

        return files.ToArray();
    }

    string LoadFile(string path)
    {
        var file = new File();
        file.Open(path, File.ModeFlags.Read);
        var content = file.GetAsText();
        file.Close();
        return content;
    }

    void LoadScripts()
    {
        var dirPath = "res://Scripts/";
        foreach (var fileName in CollectScriptFiles(dirPath))
        {
            _engine.Execute(LoadFile($"{dirPath}{fileName}"));
            GD.Print($"Script {fileName} loaded.");
        }
    }

    void Run()
    {
        var mainClass = _engine.Runtime.Globals.GetVariable("Main");
        _main = _engine.Operations.CreateInstance(mainClass, _systemAPI, _primitivesAPI);
    }
}
