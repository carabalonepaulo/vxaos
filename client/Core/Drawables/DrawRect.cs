using Godot;

namespace RGSSPlayer.Drawables
{
    public class DrawRect : Drawable
    {
        Color _color;

        public DrawRect(Rect2 rect, Color color)
        {
            DestRect = rect;
            _color = color;
        }

        public override Color GetPixel(int x, int y) => _color;

        public override void Draw(Godot.Node2D node) => node.DrawRect(DestRect, _color);
    }
}