using Godot;
using System;
using Microsoft.Scripting.Hosting;

namespace API
{
    public interface ISystem
    {
        RID root { get; }
        void log(string color, string mode, string line);
        void print(string message);
        void resize(int width, int height, bool center);
    }

    public class SignalHandler : Node
    {
        ScriptEngine _engine;

        public SignalHandler(ScriptEngine engine) => _engine = engine;

        public void OnSignalReceived(Godot.Collections.Array bindings)
        {
            object target = bindings[0];
            string method = bindings[1] as string;

            _engine.Operations.InvokeMember(target, method);
        }
    }

    public class System : ISystem
    {
        public event Action<string> Error;

        RID _root;
        Node _rootNode;
        ManagedSlots<Node> _nodes;
        Viewport _rootViewport;
        RichTextLabel _statusLabel;
        SignalHandler _signalHandler;

        public RID root => _root;

        public System(Node rootNode, RID root, Viewport viewport, RichTextLabel statusLabel, SignalHandler signalHandler)
        {
            _rootNode = rootNode;
            _nodes = new ManagedSlots<Node>(1024);
            _root = root;
            _rootViewport = viewport;
            _statusLabel = statusLabel;
            _signalHandler = signalHandler;
        }

        // public int this[Node node]
        // {
        //     get {
        //         if _
        //      }
        //     set { }
        // }

        public void add_child(Node node)
        {
        }

        public void remove_child(Node node)
        {

        }

        public string get_executable_path() => OS.GetExecutablePath();

        public void log(string color, string mode, string line)
        {
            _statusLabel.AppendBbcode($"[color={color}][[/color][color=#ababab]{mode}[/color][color={color}]][/color] {line}\n");
        }

        public void print(string message)
        {
            log("#1E92F7", "print", message);
            GD.Print(message);
        }

        public void resize(int width, int height, bool center)
        {
            OS.WindowSize = new Vector2(width, height);
            _rootViewport.SetSizeOverride(true, new Vector2(width, height));
            if (center) OS.CenterWindow();
            OS.WindowBorderless = false;
        }

        public void raise(string message) => Error?.Invoke(message);

        public void connect(Node source, string signal, object listener, string method)
        {
            source.Connect(signal, _signalHandler, "OnSignalReceived", new Godot.Collections.Array() { listener, method });
        }

        public void create_instance(string objectName)
        {

        }
    }
}