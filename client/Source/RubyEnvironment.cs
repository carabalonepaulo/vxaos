using Godot;
using System;
using Microsoft.Scripting.Hosting;

public class RubyEnvironment
{
    public const string ScriptsDirPath = "res://Scripts/";
    public event Action<string, string, string[]> ExceptionRaised;

    object _main;
    Node _root;
    ScriptEngine _engine;
    ScriptRuntime _runtime;

    API.System _system;
    API.FileSystem _fileSystem;

    public static ScriptEngine Engine { get; private set; }

    public RubyEnvironment(Node root)
    {
        _root = root;
        _runtime = IronRuby.Ruby.CreateRuntime();
        _engine = _runtime.GetEngine("Ruby");

        Engine = _engine;

        LoadScripts();
        InitAPIs();
    }

    string[] ScriptList => new string[] {
        "emitter",

        "controls/control",
        "controls/button",

        "primitives/color",
        "primitives/rect",
        "primitives/vector2",

        "rgss/bitmap",
        "rgss/sprite",

        "main",
    };

    string ReadFile(string path)
    {
        var file = new File();
        file.Open(path, File.ModeFlags.Read);
        var content = file.GetAsText();
        file.Close();
        return content;
    }

    void InitAPIs()
    {
        _system = new API.System(_root);
        _system.ExceptionRaised += (message, inspect, backtrace) => ExceptionRaised?.Invoke(message, inspect, backtrace);
        _fileSystem = new API.FileSystem();
    }

    public void LoadScripts()
    {
        foreach (var partialFileName in ScriptList)
            _engine.Execute(ReadFile(ScriptsDirPath + partialFileName + ".rb"));
    }

    public void Run()
    {
        var mainClass = RubyEnvironment.Engine.Runtime.Globals.GetVariable("Main");
        _main = RubyEnvironment.Engine.Operations.CreateInstance(mainClass, _system, _fileSystem);
    }

    public void Notification(int what) => _engine.Operations.InvokeMember(_main, "notification", what);

    public void Process(float dt) => _engine.Operations.InvokeMember(_main, "process", dt);
}