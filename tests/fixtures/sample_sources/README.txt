
# simple_module.py
def func_one(a, b):
    """Docstring for func_one."""
    pass

def func_two():
    pass

# class_heavy_module.py
class MyClass:
    """Class docstring."""
    def __init__(self, x):
        self.x = x
        
    def method_one(self):
        pass
        
    def _private(self):
        pass

def standalone_func():
    pass

# private_members.py
def _private_func():
    pass

class _PrivateClass:
    pass

def public_func():
    pass

# async_module.py
async def async_func(x):
    pass

class AsyncClass:
    async def async_method(self):
        pass

# empty_module.py
# Just a comment

# syntax_error_module.py
def broken_func(
