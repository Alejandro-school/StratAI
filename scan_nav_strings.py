import os
import struct
import re

def find_strings(filepath):
    try:
        with open(filepath, 'rb') as f:
            data = f.read()
            
        print(f"File size: {len(data)} bytes")
        
        # Print any printable string > 4 chars
        print("Scanning for any printable strings > 4 chars...")
        chars = []
        for byte in data:
            if 32 <= byte <= 126:
                chars.append(chr(byte))
            else:
                if len(chars) > 4:
                    s = "".join(chars)
                    # Filter out noise
                    if re.match(r'^[A-Za-z0-9_]+$', s):
                        print(f"String: '{s}'")
                chars = []

        
        # Check the beginning of the file for these strings
        # The header is usually ~20-30 bytes.
        # If strings are at 0x20, they are at the start.
        
    except FileNotFoundError:
        print(f"File not found: {filepath}")

if __name__ == "__main__":
    # Adjust path as needed for the workspace
    path = "backend/data/maps/de_nuke/de_nuke.nav"
    if not os.path.exists(path):
        # Try absolute path if relative fails
        path = r"e:\Carpeta compartida\Proyecto IA\Proyecto IA\backend\data\maps\de_nuke\de_nuke.nav"
    find_strings(path)
