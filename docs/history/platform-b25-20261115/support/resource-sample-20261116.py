import ctypes
from ctypes import wintypes
import json
import time

class Memory(ctypes.Structure):
    _fields_ = [('length', wintypes.DWORD), ('load', wintypes.DWORD),
                ('physical', ctypes.c_ulonglong), ('free_physical', ctypes.c_ulonglong),
                ('pagefile', ctypes.c_ulonglong), ('free_pagefile', ctypes.c_ulonglong),
                ('virtual', ctypes.c_ulonglong), ('free_virtual', ctypes.c_ulonglong),
                ('extended', ctypes.c_ulonglong)]

def times():
    values = [wintypes.FILETIME() for _ in range(3)]
    assert ctypes.windll.kernel32.GetSystemTimes(*(ctypes.byref(v) for v in values))
    return [(v.dwHighDateTime << 32) | v.dwLowDateTime for v in values]

before = times()
time.sleep(1)
after = times()
idle, kernel, user = [b - a for a, b in zip(before, after)]
memory = Memory(length=ctypes.sizeof(Memory))
assert ctypes.windll.kernel32.GlobalMemoryStatusEx(ctypes.byref(memory))
print(json.dumps({'cpu_percent': round(100 * (1 - idle / (kernel + user)), 1),
                  'memory_load_percent': memory.load, 'free_ram_bytes': memory.free_physical}))
