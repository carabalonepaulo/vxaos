using System;
using Godot;
using System.Reflection;
using System.Linq;
using System.Collections.Generic;
using IronRuby.Builtins;

namespace API
{
    public interface IRubyControl { }

    public interface IBucket
    {
        Type this[string name] { get; }
    }

    public class ControlBucket : IBucket
    {
        Dictionary<string, Type> _cache;

        public ControlBucket()
        {
            _cache = new Dictionary<string, Type>();
            foreach (var type in FindTypes())
                _cache.Add(type.Name, type);
        }

        IEnumerable<Type> FindTypes() => Assembly.GetAssembly(typeof(API.Controls.Control))
            .GetTypes()
            .Where(t => t.GetInterface("IRubyControl") != null);

        public Type this[string name] { get => _cache[name]; }
    }

    public class GodotObjectBucket : IBucket
    {
        Dictionary<string, Type> _cache;

        public GodotObjectBucket()
        {
            _cache = new Dictionary<string, Type>();
        }

        Type FindType(string name)
        {
            if (_cache.ContainsKey(name)) return _cache[name];
            var type = Assembly.GetAssembly(typeof(Godot.Node))
                .GetTypes()
                .Where(t => t.Name == name)
                .First();
            _cache[name] = type;
            return type;
        }

        public Type this[string name] { get => FindType(name); }
    }

    public class System
    {
        public event Action<string, string, string[]> ExceptionRaised;

        ControlBucket _controlBucket;
        GodotObjectBucket _godotObjectBucket;
        List<string> _errorLines;

        public readonly Godot.Node Root;

        public System(Godot.Node root)
        {
            Root = root;
            _controlBucket = new ControlBucket();
            _godotObjectBucket = new GodotObjectBucket();
        }

        public void Print(object message) => GD.Print(message.ToString());

        public void RaiseException(string message, string inspect, RubyArray backtrace)
        {
            ExceptionRaised?.Invoke(message, inspect, backtrace.Select(line => line.ToString()).ToArray());
        }

        public object CreateControl(string name) => Activator.CreateInstance(_controlBucket[name]);

        public object CreateGodotObject(string name, params object[] args) => Activator
            .CreateInstance(_godotObjectBucket[name], args);

        public void Quit() { Root.GetTree().Quit(); }
    }
}