using Godot;
using System;
using System.Linq;
using System.Reflection;

namespace API
{
    public class Primitives
    {
        public Vector2 Vector2(float x, float y) => new Vector2(x, y);

        public Color Color(byte r, byte g, byte b, byte a = 255)
        {
            return Godot.Color.Color8(r, g, b, a);
        }

        public Rect2 Rect(float x, float y, float width, float height)
        {
            return new Godot.Rect2(x, y, width, height);
        }

        public Transform2D Transform(float xx, float xy, float yx, float yy, float ox, float oy)
        {
            return new Transform2D(xx, xy, yx, yy, ox, oy);
        }

        public Node Node(string name)
        {
            var type = Assembly.GetAssembly(typeof(Node))
                .GetTypes()
                .Where(t => t.FullName == name)
                .First();

            return Activator.CreateInstance(type) as Node;
        }
    }
}