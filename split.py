import re
import os

with open('linq.go', 'r') as f:
    text = f.read()

# Match top level functions, structs, types.
# A simplified parser that looks for ^(type|func) ... { and captures until matched }.
blocks = []
lines = text.split('\n')
current_block = []
in_block = False
brace_count = 0
block_name = ""

header = []

for line in lines:
    if not in_block:
        if line.startswith('package ') or line.startswith('import ') or (line.startswith('//') and block_name == "") or (not line.strip() and block_name == ""):
            if not line.startswith('func ') and not line.startswith('type '):
                header.append(line)
        
        m = re.match(r'^(?://.*?\n)*?(func|type)\s+(?:\([^\)]+\)\s+)?([A-Za-z0-9_\[\]\*,]+)', line)
        if m or line.startswith('func ') or line.startswith('type '):
            in_block = True
            current_block = [line]
            brace_count += line.count('{') - line.count('}')
            if brace_count == 0 and ('{' not in line):
                # single line type or func definition without braces? e.g. type ... func(...)
                if ' struct ' not in line and ' interface ' not in line and not line.rstrip().endswith('{'):
                    in_block = False
                    # grab preceding comments
                    # handled manually...
            continue
    else:
        current_block.append(line)
        brace_count += line.count('{') - line.count('}')
        if brace_count <= 0:
            blocks.append('\n'.join(current_block))
            in_block = False
            current_block = []

print(f"Found {len(blocks)} chunks to ignore, parser is not robust.")
