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
    API.FileSystem _fileSystemAPI;

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
        _fileSystemAPI = new API.FileSystem();
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

    string[] CollectScriptFiles(string rootDir)
    {
        var files = new List<string>();
        var dir = new Directory();
        dir.Open(rootDir);
        dir.ListDirBegin(true, true);

        var fileName = dir.GetNext();
        var innerFileList = new List<string>();

        while (fileName != "")
        {
            var filePath = dir.GetCurrentDir() + "/" + fileName;
            if (dir.CurrentIsDir()) innerFileList.AddRange(CollectScriptFiles(filePath));
            else files.Add(filePath);
            // if (dir.CurrentIsDir()) files.AddRange(CollectScriptFiles(filePath));
            // else files.Add(filePath);
            fileName = dir.GetNext();
        }
        files.AddRange(innerFileList);
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
        var files = CollectScriptFiles("res://Scripts/");
        foreach (var filePath in files)
            _engine.Execute(LoadFile(filePath));
        GD.Print($"{files.Length} scripts successfully loaded.");
    }

    void Run()
    {
        var mainClass = _engine.Runtime.Globals.GetVariable("Main");
        _main = _engine.Operations.CreateInstance(mainClass, _systemAPI, _fileSystemAPI);
    }
}
