using Godot;

namespace RGSSPlayer.Drawables
{
    public class DrawString : Drawable
    {
        string _text;
        int _align;
        DynamicFont _font;
        Color _color;

        public DrawString(Rect2 rect, string text, int align, DynamicFont font, Color color)
        {
            DestRect = rect;
            _text = text;
            _align = align;
            _font = font;
            _color = color;
        }

        public override Color GetPixel(int x, int y) => _color;

        public override void Draw(Godot.Node2D node) => node.DrawString(_font, DestRect.Position, _text, _color, (int)DestRect.Size.x);
    }
}