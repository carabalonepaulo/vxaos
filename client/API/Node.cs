using Godot;
using System;
using System.Collections.Generic;
using System.Reflection;

namespace API
{
    public interface IEmitter
    {
        object Emitter { get; set; }
    }

    public class Node : RubyAPI
    {
        public event Action<string> ExceptionRaised;

        Dictionary<string, Type> _nodes;

        public Node()
        {
            _nodes = new Dictionary<string, Type>()
            {
                { "Control", typeof(API.Nodes.Control) },
                { "HorizontalBox", typeof(API.Nodes.HorizontalBox) },
                { "VerticalBox", typeof(API.Nodes.VerticalBox) },
                { "Button", typeof(API.Nodes.Button) },
                { "LineEdit", typeof(API.Nodes.LineEdit) }
            };
        }

        public Godot.Node CreateNode(string name)
        {
            if (!_nodes.ContainsKey(name))
            {
                ExceptionRaised?.Invoke($"Can't create '{name}' node.");
                return null;
            }
            return Activator.CreateInstance(_nodes[name]) as Godot.Node;
        }
    }
}