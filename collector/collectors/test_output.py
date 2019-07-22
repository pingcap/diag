# coding:utf8
import unittest
from pyfakefs.fake_filesystem_unittest import Patcher


from output import FileOutput


class TestFileOutput(unittest.TestCase):
    def test_output(self):
        with Patcher():
            filename = 'test.data'
            out = FileOutput(filename)
            out.output('hello')
            with open(filename) as f:
                self.assertEqual(f.read(), 'hello')


if __name__ == "__main__":
    unittest.main()
