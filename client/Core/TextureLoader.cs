using System;
using Godot;

public interface ITextureLoader
{
    Texture Load(string path);
}

public class LocalFSTextureLoader : ITextureLoader
{
    public Texture Load(string path)
    {
        var image = new Image();
        image.LoadPngFromBuffer(System.IO.File.ReadAllBytes(path));
        var tex = new ImageTexture();
        tex.CreateFromImage(image);
        image.Dispose();
        return tex;
    }
}

public class VirtualFSTextureLoader : ITextureLoader
{
    public Texture Load(string path) => ResourceLoader.Load<Texture>(path);
}

public class TextureLoaderFactory
{
    public ITextureLoader CreateTextureLoader()
    {
        if (Main.LocalFileSystem) return new LocalFSTextureLoader();
        return new VirtualFSTextureLoader();
    }
}