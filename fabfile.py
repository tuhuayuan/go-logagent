from __future__ import with_statement
from fabric.api import local, settings, abort, run, cd, env, put, lcd

def build(os='darwin', arch='amd64', binary='logagent',):
    with lcd('./cmd'):
        targetPath = "/".join(['../build', os, arch])
        target = targetPath + '/' + binary
        local("mkdir -p " + targetPath)
        local('GOOS='+os+' GOARCH='+arch+' go build -o '+target)