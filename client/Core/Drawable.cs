using Godot;

namespace RGSSPlayer
{
    public abstract class Drawable
    {
        public Rect2 DestRect { get; protected set; }

        public abstract Color GetPixel(int x, int y);

        public abstract void Draw(Godot.Node2D node);

        public bool Intersect(int x, int y)
        {
            return x >= DestRect.Position.x
                && x <= DestRect.Position.x + DestRect.Size.x
                && y >= DestRect.Position.y
                && y <= DestRect.Position.y + DestRect.Size.y;
        }
    }
}