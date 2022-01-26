using Godot;
using System;
using System.Collections.Generic;
using System.Reflection;
using System.Linq;

namespace API
{
    public interface IEmitter
    {
        object Emitter { get; set; }
    }

    public class Node
    {
        public Godot.Node CreateNode(string name)
        {
            var type = Assembly.GetAssembly(typeof(API.Nodes.Control))
                .GetTypes()
                .Where(t => t.Name == name)
                .First();

            return Activator.CreateInstance(type) as Godot.Node;
        }
    }
}