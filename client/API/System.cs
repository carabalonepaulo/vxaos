using System;
using Godot;

namespace API
{
    public class System : RubyAPI
    {
        public event Action<String> ExceptionRaised;

        public readonly Godot.Node Root;

        public System(Godot.Node root)
        {
            Root = root;
        }

        public void Print(string message) => GD.Print(message);

        public void RaiseException(string message) => ExceptionRaised?.Invoke(message);

        public void Quit() { }
    }
}