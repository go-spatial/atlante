var test = require('tape');
var link = require('../lib/symlink.js');
var fs = require('fs');
var path = require('path');
var rimraf = require('rimraf');
var fse = require('fs-extra');
var appDir = process.cwd();
var sinon = require('sinon');
var log = require('npmlog');

global.appRoot = process.cwd();

function setupSymlinks(filePaths, callback){
  var firstSymSource = path.join(__dirname + '/fixtures/fake', 'symlink/');
  var secondSymSource = path.join(__dirname + '/fixtures/fake', 'symlink-copy/');

  fse.mkdirpSync(firstSymSource);
  fse.mkdirpSync(secondSymSource);
  fse.mkdirpSync(__dirname + '/fixtures/fake');

  fs.symlinkSync(secondSymSource, filePaths[0][0]);
  fs.symlinkSync(firstSymSource, filePaths[0][1]);
  return callback(null);
}

function cleanUpSymlinks(callback){
  // fse.removeSync(firstSymSource);
  // fse.removeSync(secondSymSource);
  fse.removeSync(__dirname + '/fixtures/fake');
  return callback(null);
}

test('setup', function(assert) {

  if (fs.existsSync(__dirname + '/fixtures/out/mason_packages/.link')) fse.removeSync(__dirname + '/fixtures/out/mason_packages/.link');
  fse.mkdirpSync(__dirname + '/fixtures/out/mason_packages/.link');

  assert.end();
});

test('[symlink] builds correct symlink paths', function(assert) {
  var symlinkPath = path.join(global.appRoot, 'test/fixtures/out/mason_packages/.link');

  var libraries = [{
    name: 'protozero',
    version: '1.5.1',
    headers: true,
    os: null,
    awsPath: 'headers/protozero/1.5.1.tar.gz',
    src: 'https://s3.amazonaws.com/mason-binaries/headers/protozero/1.5.1.tar.gz',
    dst: appDir + '/test/fixtures/headers/protozero/1.5.1'
  },
  {
    name: 'cairo',
    version: '1.14.8',
    headers: null,
    os: 'osx-x86_64',
    awsPath: 'osx-x86_64/cairo/1.14.8.tar.gz',
    src: 'https://s3.amazonaws.com/mason-binaries/osx-x86_64/cairo/1.14.8.tar.gz',
    dst: appDir + '/test/fixtures/osx-x86_64/cairo/1.14.8'
  }];

  var expected = [
    [appDir + '/test/fixtures/headers/protozero/1.5.1',
      symlinkPath
    ],
    [appDir + '/test/fixtures/osx-x86_64/cairo/1.14.8',
      symlinkPath
    ]
  ];

  var result = link.buildLinkPaths(libraries, symlinkPath);
  assert.deepEqual(result, expected);
  assert.end();
});

test('[symlink] creates symlink', function(assert) {
  var symlinkPath = path.join(global.appRoot, 'test/fixtures/out/mason_packages/.link');

  var paths = [
    [appDir + '/test/fixtures/headers/protozero/1.5.1',
      symlinkPath
    ],
    [appDir + '/test/fixtures/osx-x86_64/cairo/1.14.8',
      symlinkPath
    ]
  ];

  var proto = path.join(appDir + '/test/fixtures/out', 'mason_packages/.link', 'include', 'protozero', 'byteswap.hpp');
  var cairo = path.join(appDir + '/test/fixtures/out', 'mason_packages/.link', 'include', 'cairo', 'cairo-ft.h');

  sinon.spy(log, 'info');

  link.symLink(paths, function(err, result) {
    assert.equal(result, true);
    assert.equal(fs.existsSync(proto), true);
    assert.equal(fs.existsSync(cairo), true);
    assert.equal(log.info.getCall(0).args[0], 'Symlinked: ');
    assert.end();
  });
});

test('[symlink] fails to create symlink - directory not found', function(assert) {
  var symlinkPath = path.join(global.appRoot, 'test/fixtures/out/mason_packages/.link');

  var paths = [
    [appDir + '/test/fixtures/headers/protozro/1.5.1/',
      symlinkPath
    ],
    [appDir + '/test/fixtures/osx-x8_64/cairo/1.14.8/',
      symlinkPath
    ]
  ];

  link.symLink(paths, function(err) {
    assert.equal(/ENOENT: no such file or directory/.test(err.message), true);
    assert.end();
  });
});

test('[symlink] overwrites existing files', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'config.cpp');
  var dst = path.join(__dirname + '/fixtures/', 'config-copy.cpp');

  var paths = [
    [src,
      dst
    ]
  ];

  link.symLink(paths, function(err, result) {
    assert.equal(err, null);
    assert.equal(result, true);
    assert.end();
  });
});

test('[symlink] overwrites existing destination symlink with symlink source', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'fake', 'temp');
  var dst = path.join(__dirname + '/fixtures/', 'fake', 'tmp');

  var paths = [
    [src, dst]
  ];

  sinon.spy(fs, 'existsSync');
  sinon.spy(fse, 'removeSync');

  setupSymlinks(paths, function(){
    var lsync = fs.lstatSync(path.join(__dirname + '/fixtures/', 'fake', 'temp'));
    assert.equal(lsync.isSymbolicLink(), true, 'src is symbolic link');

    link.symLink(paths, function(err, result) {
      assert.equal(err, null);
      assert.equal(result, true);
      assert.equal(fse.removeSync.calledOnce, true, 'remove sync called once');
      assert.equal(fs.existsSync.calledOnce, true);

      cleanUpSymlinks(function(){
        fs.existsSync.restore();
        fse.removeSync.restore();
        assert.end();
      });
    });
  });
});

test('[symlink] doesnt symlink mason.ini files', function(assert) {
  var src = '/test/fixtures/headers/protozro/1.5.1/mason.ini';
  var dest = 'symlink/path';

  var filter = link.filterFunc(src, dest);

  assert.equal(filter, false);
  assert.end();
});

test('cleanup', (assert) => {
  rimraf(__dirname + '/fixtures/out', function(err) {
    assert.ifError(err);
    assert.end();
  });
});
