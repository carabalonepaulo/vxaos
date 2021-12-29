using Godot;

namespace API
{
    public interface IPrimitive
    {
        Vector2 vector2(float x, float y);
        Color color(byte r, byte g, byte b, byte a = 255);
        Rect2 rect(float x, float y, float width, float height);
        Transform2D transform(float xx, float xy, float yx, float yy, float ox, float oy);
        Texture texture(string path);
    }

    public class Primitives : IPrimitive
    {
        public Vector2 vector2(float x, float y) => new Vector2(x, y);

        public Color color(byte r, byte g, byte b, byte a = 255)
        {
            return Godot.Color.Color8(r, g, b, a);
        }

        public Rect2 rect(float x, float y, float width, float height)
        {
            return new Godot.Rect2(x, y, width, height);
        }

        public Transform2D transform(float xx, float xy, float yx, float yy, float ox, float oy)
        {
            return new Transform2D(xx, xy, yx, yy, ox, oy);
        }

        public Texture texture(string path) => ResourceLoader.Load<Texture>(path);
    }
}