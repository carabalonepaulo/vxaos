using System;
using System.IO;
using Godot;
using System.Text;

public class OutputStream : Stream
{
    MemoryStream _stream;
    public OutputStream()
    {
        _stream = new MemoryStream();
    }

    public override bool CanRead => false;

    public override bool CanSeek => false;

    public override bool CanWrite => true;

    public override long Length => -1;

    public override long Position { get => -1; set => throw new NotImplementedException(); }

    public override void Flush() => throw new NotImplementedException();

    public override int Read(byte[] buffer, int offset, int count) => throw new NotImplementedException();

    public override long Seek(long offset, SeekOrigin origin) => throw new NotImplementedException();

    public override void SetLength(long value) => throw new NotImplementedException();

    public override void Write(byte[] buffer, int offset, int count) => GD.Print(Encoding.UTF8.GetString(buffer));
}