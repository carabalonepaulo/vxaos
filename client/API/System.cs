using System;
using Godot;
using System.Reflection;
using System.Linq;
using System.Collections.Generic;

namespace API
{
    public interface IRubyControl { }

    public class NodeBucket
    {
        Dictionary<string, Type> _cache;

        public NodeBucket()
        {
            _cache = new Dictionary<string, Type>();
            foreach (var type in FindTypes())
                _cache.Add(type.Name, type);
        }

        IEnumerable<Type> FindTypes() => Assembly.GetAssembly(typeof(API.Nodes.Control))
            .GetTypes()
            .Where(t => t.GetInterface("IRubyControl") != null);

        public Type this[string name] { get => _cache[name]; }
    }

    public class GodotObjectBucket
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

    public class System : RubyAPI
    {
        public event Action<String> ExceptionRaised;

        NodeBucket _nodeBucket;
        GodotObjectBucket _godotObjectBucket;

        public readonly Godot.Node Root;

        public System(Godot.Node root)
        {
            Root = root;
            _nodeBucket = new NodeBucket();
            _godotObjectBucket = new GodotObjectBucket();
        }

        public void Print(string message) => GD.Print(message);

        public void RaiseException(string message) => ExceptionRaised?.Invoke(message);

        public object CreateNode(string name) => Activator.CreateInstance(_nodeBucket[name]);

        public object CreateGodotObject(string name, params object[] args) => Activator
            .CreateInstance(_godotObjectBucket[name], args);

        public void Quit() { Root.GetTree().Quit(); }
    }
}