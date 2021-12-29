using Godot;

namespace RGSSPlayer.Drawables
{
    public class DrawTextureRectRegion : Drawable
    {
        Texture _texture;
        Rect2 _sourceRect;
        byte _opacity;

        public DrawTextureRectRegion(Rect2 destRect, Texture texture, Rect2 srcRect, byte opacity)
        {
            DestRect = destRect;
            _texture = texture;
            _sourceRect = srcRect;
            _opacity = opacity;
        }

        Color Color => Color.Color8(255, 255, 255, _opacity);

        // TODO: not working properly, only works with non deformed textures
        public override Color GetPixel(int x, int y) => _texture.GetData().GetPixel(x, y);

        public override void Draw(Godot.Node2D node) => node.DrawTextureRectRegion(_texture, DestRect, _sourceRect, Color);
    }
}