using Godot;
using System;
using Microsoft.Scripting.Hosting;
using System.CodeDom;
using IronRuby.Runtime;
using IronRuby;

public class RubyEnvironment
{
    public const string ScriptsDirPath = "res://Scripts/";
    public event Action<string, string, string[]> ExceptionRaised;

    object _main;
    Node _root;
    ScriptEngine _engine;
    ScriptRuntime _runtime;
    RubyContext _context;
    OutputStream _stream;

    API.System _system;
    API.FileSystem _fileSystem;

    public static ScriptEngine Engine { get; private set; }

    public RubyEnvironment(Node root)
    {
        _root = root;
        _runtime = IronRuby.Ruby.CreateRuntime();
        _engine = _runtime.GetEngine("Ruby");
        _stream = new OutputStream();
        _runtime.IO.SetOutput(_stream, System.Text.Encoding.UTF8);
        _runtime.LoadAssembly(typeof(Godot.GD).Assembly);

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
        "nodes/aspect_ratio_container",
        "nodes/box_container",
        "nodes/color_picker",
        "nodes/hbox_container",
        "nodes/vbox_container",
        "nodes/center_container",
        "nodes/scroll_container",
        "nodes/graph_node",
        "nodes/grid_container",
        "nodes/split_container",
        "nodes/hsplit_container",
        "nodes/vsplit_container",
        "nodes/margin_container",
        "nodes/panel_container",
        "nodes/tab_container",
        "nodes/viewport_container",
        "nodes/base_button",
        "nodes/button",
        "nodes/check_box",
        "nodes/check_button",
        "nodes/color_picker_button",
        "nodes/menu_button",
        "nodes/option_button",
        "nodes/tool_button",
        "nodes/link_button",
        "nodes/texture_button",
        "nodes/color_rect",
        "nodes/range",
        "nodes/scroll_bar",
        "nodes/hscroll_bar",
        "nodes/vscroll_bar",
        "nodes/progress_bar",
        "nodes/spin_box",
        "nodes/texture_progress",
        "nodes/separator",
        "nodes/hseparator",
        "nodes/vseparator",
        "nodes/item_list",
        "nodes/label",
        "nodes/line_edit",
        "nodes/nine_patch_rect",
        "nodes/panel",
        "nodes/reference_rect",
        "nodes/rich_text_label",

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