import ctypes
library = ctypes.cdll.LoadLibrary('./clib/library.so')
hello_world = library.helloWorld
hello_world()