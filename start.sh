#!/bin/bash
if [[ "$OSTYPE" == "linux-gnu"* ]]; then # comando para LINUX

  konsole & sleep 1

  qdbus org.kde.konsole-$! /konsole/MainWindow_1 org.kde.KMainWindow.activateAction split-view-left-right

  qdbus org.kde.konsole-$! /Windows/1 org.kde.konsole.Window.setCurrentSession 1
  qdbus org.kde.konsole-$! /Sessions/1 setTitle 1 'FRONTEND'
  qdbus org.kde.konsole-$! /Sessions/1 runCommand 'cd ./frontend && npm run dev'

  qdbus org.kde.konsole-$! /Sessions/2 setTitle 1 'BACKEND'
  qdbus org.kde.konsole-$! /Sessions/2 runCommand 'cd ./backend && air dev'

  qdbus org.kde.konsole-$! /Windows/2 org.kde.konsole.Window.setCurrentSession 1
  qdbus org.kde.konsole-$! /Sessions/0 setTitle 1 'GERP'

else # comando para WINDOWS

   echo 'nada configurado'
   echo 'probar cmder ==> https://www.eventslooped.com/posts/automate-open-tabs-in-cmder/'

fi
