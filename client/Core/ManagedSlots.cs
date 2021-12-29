using System.Collections.Generic;

public class ManagedSlots<T>
{
    int _capacity;
    T[] _slots;
    Stack<int> _availableIndices;
    int _highestIndex = -1;

    public ManagedSlots(int capacity)
    {
        _capacity = capacity;
        _slots = new T[capacity];
        _availableIndices = new Stack<int>();
    }

    public int GetFreeId()
    {
        if (_availableIndices.Count > 0)
            return _availableIndices.Pop();
        _highestIndex += 1;
        return _highestIndex;
    }

    public void Free(int id)
    {
        _availableIndices.Push(id);
        _slots[id] = default(T);
    }

    public T this[int id]
    {
        get { return _slots[id]; }
        set { _slots[id] = value; }
    }
}
