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
    dynamic _main;
    bool _hasError;
    ErrorDialog _errorDialog;
    RubyEnvironment _env;

    public override void _Ready()
    {
        SetupErrorDialog();
        SetupRubyEnvironment();
        // RaiseException(new string[] { "line 1", "line 2", "line 3", "line 4" });
    }

    public override void _Notification(int what)
    {
        if (_hasError || _main == null) return;
        _env.Notification(what);
    }

    public override void _Process(float delta)
    {
        if (_hasError || _main == null) return;
        _env.Process(delta);
    }

    void SetupErrorDialog() => _errorDialog = GetNode<ErrorDialog>("ErrorDialog");

    void OnDialogConfirmed() => GetTree().Quit();

    void RaiseException(string message, string inspect, string[] backtrace)
    {
        _hasError = true;
        _errorDialog.RaiseException(message, inspect, backtrace);
    }

    void SetupRubyEnvironment()
    {
        _env = new RubyEnvironment(this);
        _env.ExceptionRaised += RaiseException;
        _env.Run();
    }
}
