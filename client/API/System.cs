using System;
using Godot;
using System.Reflection;
using System.Linq;

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

        public void Error(string message) => GD.PushError(message);

        public void RaiseException(string message) => ExceptionRaised?.Invoke(message);

        public object CreateInstance(string fullName, params object[] args)
        {
            var type = Assembly.GetExecutingAssembly()
                .GetTypes()
                .Where(t => t.FullName == fullName)
                .First();
            return Activator.CreateInstance(type, args);
        }

        public void Quit() { }
    }
}