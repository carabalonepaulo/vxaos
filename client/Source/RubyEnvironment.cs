using Godot;
using System;
using Microsoft.Scripting.Hosting;
using System.CodeDom;

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
        "wrapper",

        "nodes/node",
        "nodes/canvas_item",
        "nodes/control",
        "nodes/container",
        "nodes/base_button",
        "nodes/button",

        "godot_objects/color",
        "godot_objects/rect",
        "godot_objects/vector2",
        "godot_objects/node_path",
        "godot_objects/scene_tree",
        "godot_objects/viewport",

        "rgss/bitmap",
        "rgss/sprite",

        "rgss",
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
        var mainClass = _engine.Runtime.Globals.GetVariable("Main");
        _main = _engine.Operations.CreateInstance(mainClass, _system, _fileSystem);
    }

    public void Notification(int what) => _engine.Operations.InvokeMember(_main, "notification", what);

    public void Process(float dt) => _engine.Operations.InvokeMember(_main, "process", dt);
}