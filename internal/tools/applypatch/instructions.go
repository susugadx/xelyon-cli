package applypatch

const applyPatchDescription = "## `apply_patch`\n\n" +
	"Use the `apply_patch` shell command to edit files.\n" +
	"Your patch language is a stripped-down, file-oriented diff format designed to be easy to parse and safe to apply. You can think of it as a high-level envelope:\n\n" +
	"*** Begin Patch\n" +
	"[ one or more file sections ]\n" +
	"*** End Patch\n\n" +
	"Within that envelope, you get a sequence of file operations.\n" +
	"You MUST include a header to specify the action you are taking.\n" +
	"Each operation starts with one of three headers:\n\n" +
	"*** Add File: <path> - create a new file. Every following line is a + line (the initial contents).\n" +
	"*** Delete File: <path> - remove an existing file. Nothing follows.\n" +
	"*** Update File: <path> - patch an existing file in place (optionally with a rename).\n\n" +
	"May be immediately followed by *** Move to: <new path> if you want to rename the file.\n" +
	"Then one or more \"hunks\", each introduced by @@ (optionally followed by a hunk header).\n" +
	"Within a hunk each line starts with:\n\n" +
	"For instructions on [context_before] and [context_after]:\n" +
	"- By default, show 3 lines of code immediately above and 3 lines immediately below each change. If a change is within 3 lines of a previous change, do NOT duplicate the first change's [context_after] lines in the second change's [context_before] lines.\n" +
	"- If 3 lines of context is insufficient to uniquely identify the snippet of code within the file, use the @@ operator to indicate the class or function to which the snippet belongs. For instance, we might have:\n" +
	"@@ class BaseClass\n" +
	"[3 lines of pre-context]\n" +
	"- [old_code]\n" +
	"+ [new_code]\n" +
	"[3 lines of post-context]\n\n" +
	"- If a code block is repeated so many times in a class or function such that even a single `@@` statement and 3 lines of context cannot uniquely identify the snippet of code, you can use multiple `@@` statements to jump to the right context. For instance:\n\n" +
	"@@ class BaseClass\n" +
	"@@ \t def method():\n" +
	"[3 lines of pre-context]\n" +
	"- [old_code]\n" +
	"+ [new_code]\n" +
	"[3 lines of post-context]\n\n" +
	"The full grammar definition is below:\n" +
	"Patch := Begin { FileOp } End\n" +
	"Begin := \"*** Begin Patch\" NEWLINE\n" +
	"End := \"*** End Patch\" NEWLINE\n" +
	"FileOp := AddFile | DeleteFile | UpdateFile\n" +
	"AddFile := \"*** Add File: \" path NEWLINE { \"+\" line NEWLINE }\n" +
	"DeleteFile := \"*** Delete File: \" path NEWLINE\n" +
	"UpdateFile := \"*** Update File: \" path NEWLINE [ MoveTo ] { Hunk }\n" +
	"MoveTo := \"*** Move to: \" newPath NEWLINE\n" +
	"Hunk := \"@@\" [ header ] NEWLINE { HunkLine } [ \"*** End of File\" NEWLINE ]\n" +
	"HunkLine := (\" \" | \"-\" | \"+\") text NEWLINE\n\n" +
	"A full patch can combine several operations:\n\n" +
	"*** Begin Patch\n" +
	"*** Add File: hello.txt\n" +
	"+Hello world\n" +
	"*** Update File: src/app.py\n" +
	"*** Move to: src/main.py\n" +
	"@@ def greet():\n" +
	"-print(\"Hi\")\n" +
	"+print(\"Hello, world!\")\n" +
	"*** Delete File: obsolete.txt\n" +
	"*** End Patch\n\n" +
	"It is important to remember:\n\n" +
	"- You must include a header with your intended action (Add/Delete/Update)\n" +
	"- You must prefix new lines with `+` even when creating a new file\n" +
	"- File references can only be relative, NEVER ABSOLUTE.\n\n" +
	"You can invoke apply_patch like:\n\n" +
	"```\n" +
	"shell {\"command\":[\"apply_patch\",\"*** Begin Patch\\n*** Add File: hello.txt\\n+Hello, world!\\n*** End Patch\\n\"]}\n" +
	"```"
