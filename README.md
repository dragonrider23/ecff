Enhanced Configuration File Format
==================================

ECFF is a simplistic yet versatile configuration file format. It's designed to be user friendly and easy to use.

This repo contains a sample implementation of ECFF in Go using reflection. It's not complete yet nor is it optimized. It is, right now, just a quick and dirty true. I'm having a few issues with reflection (this is my first project using Go's reflection extensively) but eventually this will serve as a full implementation example.

Spec
----

Comment / Blank Lines
---------------------

Comments must be on their own line and must begin with a "#". Leading whitespace doesn't matter except inside a list in which case if the leading whitespace fits the current list it should be considered a list item.

Key Value Pairs
---------------

The simplest configuration is key value pairs in the form `key: value`.

Keys are strings mapped to may contain spaces and should have the following rules applied before storing in a data structure:

1. Convert to all lowercase
2. Capitalize each word
3. Remove spaces

The order doesn't really matter so long as end result is the same. Example: `server bind address: 127.0.0.1` would be stored as `ServerBindAddress` to `127.0.0.1`

Values may be a string, signed/unsigned int, float, or boolean. For booleans, the strings "true", "t", "yes", and "1" should be interpreted as boolean true. Likewise "false", "f", "no", and "0" are boolean false. These values should be interpreted case insensitively.

Key value pairs should be one pair per line. This way every thing after the separating semicolon should be considered the value of the key. Space between the key, semicolon, and start of value is insignificant. Leading whitespace is also insignificant.

Simple Lists
------------

There are three types of lists in ECFF. The first is a simple list with the format:

```
List1:
    Item1
    Item2
    Item3
```

Each line is considered a single element. Leading whitespace of elements matters. If a line has leading whitespace that differs from the rest, the parser may return an error or attempt parse as a new list or key-value pair if there's no whitespace.

List items must be the same type.

Named Lists
-----------

Named lists differ in that they have the same key but may be indexed in an associative array or some other map structure using a unique name. Example:

```
List: block1
    Item 1
    Item 2
    Item 3

List: block2
    Item 1
    Item 2
    Item 3
```

These can be useful if you want multiple of the same type of list but not sure until runtime how many will be needed. Again, all items must be the same type.

Extended Lists
--------------

The last type of list is the extended list which alone with it's list of items, it can have key-value pairs as settings. Example:

```
eList: block1 setting1 = "value" setting2 = "value with space"
    Item1
    Item 2
    Item 5
```

In this example the main struct key is "eList", the name of the list is "block1" and it has two settings "setting1" and "setting2". Settings have the format `settingName = "setting value"`. The setting name must not have a space. Whitespace around the equal sign doesn't matter. The value must be surrounded in double quotes.

Full Example
------------

```
# This is a comment
# Simple key value pairs
name: John Doe
date of employment: 10/27/2015
hourly pay: 15.21
id number 123456

# Simple list
titles:
    Head of IT
    Head of Technology
    Grand Puba of Computers

# Extended list
projects: server-room finished = "false" dateOfStart = "10/26/2015"
    Move current servers to new room
    Move fiber optic connections
    Drink coffee and rest for an hour
    Wonder why everyone is worried the network's down
    Remember you never connected the servers
    Go home for the weekend and finish it Monday
```
