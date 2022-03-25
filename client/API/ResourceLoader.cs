using System;
using Godot;

namespace API
{
    public class ResourceLoader
    {
        public bool Exists(string path) => Godot.ResourceLoader.Exists(path);

        public string[] GetDependencies(string path) => Godot.ResourceLoader.GetDependencies(path);

        public string[] GetRecognizedExtensionsForType(string type) => Godot.ResourceLoader.GetRecognizedExtensionsForType(type);

        public bool Has(string path) => Godot.ResourceLoader.Has(path);

        public bool HasCached(string path) => Godot.ResourceLoader.HasCached(path);

        public Resource Load(string path, string typeHint = "", bool noCache = false) =>
            Godot.ResourceLoader.Load(path, typeHint, noCache);
    }
}