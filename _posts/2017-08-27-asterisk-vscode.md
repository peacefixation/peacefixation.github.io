---
layout: post
title:  "Asterisk syntax highlighting extension for Visual Studio Code"
date:   2017-08-27 18:00:00:00 +1000
categories: development
---

I've started using Microsoft's new [Visual Studio Code](https://code.visualstudio.com/) text editor at work and it's pretty neat. I like it more than the Brackets editor that I was using before. It's similar, but more polished and has some excellent features like an integrated terminal and a debugger. As you might expect there is a comprehensive extension repository but once again there was no syntax highlighter for Asterisk dialplan code so I took it upon myself to fill the void and wrote one. This time around there was no existing language grammar for me to use, so it was a slightly more formidable undertaking.

If you'd like to make your own extension I'd recommend following the [extensive documentation](https://code.visualstudio.com/docs/extensions/overview) to get started, it's all explained very well and I won't repeat it here. You can `npm install` Yeoman and the VS Code Extension Generator to create the boiler plate including a howto reference which is nice to have on hand.

The extension framework is fairly simple, define the extension properties in `package.json` and create a `syntaxes/asterisk.tmLanguage` file for the TextMate grammar. A few extra features are defined in the `language-configuration.json` file, such as the comment character, character pairs that will be auto-closed and characters that can be used to surround a selection, like quote marks or braces.

Syntax highlighting extensions use the TextMate framework for language grammars, similar to Atom and Sublime and .. well, TextMate. A grammar is defined by defining a regex that will match a syntax element and then setting a scope. The scope might be `comment.quoted` or `meta.function` and depending on your current colour theme, each scope will take on a certain colour and style in your editor.

Fully fledged language extensions (i.e. intellisense, auto-correct) are much more complex, I didn't need such features for this extension.

# The Grammar

Creating the regexes for the Asterisk dialplan grammar was equal parts easy and hard. I don't know of an official definition for the grammar so I just worked with the syntax that I know, and the little documentation there is. This is further compounded by the fact that I work with an older version of Asterisk (1.6), so there is some newer syntax I'm not familiar with. In any case, I did what I could, and I will improve the extension as I can.

Matching keywords, or a variable definition, or a function call was pretty easy. It was harder to match a variable inside a quoted string inside a function call, but it was all possible, and after some trial and error I have it working quite nicely.

Here's an example of a simple match for a file import declaration like `#include extensions.conf`.

    <dict>
        <key>match</key>
        <string>^#include</string>
        <key>name</key>
        <string>keyword.control.import</string>
    </dict>

Here's an example of a much more complicated match on a variable that can contain a nested variable `${CHANNEL_${MAX_CHANNELS}}`, or a nested function `${CDR(accountcode)}`. They key to effectively writing this match was capturing the open and closing parts, then capturing the nested function and including a match on `$self` for nested variables before finally matching on the rest of the inner text with a match on anything that isn't the closing part.

    <key>VariableNested</key>
    <dict>
        <key>begin</key>
        <string>(\$\{)</string>
        <key>beginCaptures</key>
        <dict>
            <key>1</key>
            <dict>
                <key>name</key>
                <string>variable</string>
            </dict>
        </dict>
        <key>end</key>
        <string>(\})</string>
        <key>endCaptures</key>
        <dict>
            <key>1</key>
            <dict>
                <key>name</key>
                <string>variable</string>
            </dict>
        </dict>
        <key>patterns</key>
        <array>
            <dict>
                <key>include</key>
                <string>#FunctionNested</string>
            </dict>
            <dict>
                <key>include</key>
                <string>$self</string>
            </dict>
            <dict>
                <key>match</key>
                <string>[^}]</string>
                <key>name</key>
                <string>variable</string>
            </dict>
        </array>
    </dict>

A cool feature of the grammar file is the repository. You can create a repository of named matches using the `<repository>` tag. Once defined, you can include a named match in the main part of the file as required. This could be a top level match, or a sub level match. In my grammar, I define a variable, and then I include it as a stand alone variable, as a nested variable inside a quoted string, as a nested variable inside a function and also as a nested variable inside an expression. Don't repeat yourself! :)

I found the [TextMate manual](http://manual.macromates.com/en/) and the [Sublime scope naming page](https://www.sublimetext.com/docs/3/scope_naming.html) very helpful. I also dug out the colour theme definitions inside the VS Code folder to see which scopes the included themes target for highlighting. I ended up choosing some scopes that were semantically incorrect, but resulted in a better highlight across multiple themes. I don't feel particularly good about that, but at the end of the day, better highlighting means faster Asterisk dialplan development, so I compromised on correctness.

Publishing the extension was simple, but you do need to create a Visual Studio Team System (VSTS) account in order to generate a `Personal Access Token`. The [documentation](https://code.visualstudio.com/docs/extensions/publish-extension) walks you through the process. Once you have your token, publish the extension by running `vsce publish -p <token>` and you're done!

You can see the full extension code on [Github](https://github.com/peacefixation/asterisk-vscode).