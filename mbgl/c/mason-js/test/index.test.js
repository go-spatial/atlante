var test = require('tape');
var fs = require('fs');
var path = require('path');
var sinon = require('sinon');
var needle = require('needle');
var stream = require('stream');
var log = require('npmlog');
var fse = require('fs-extra');
var index = require('../');
var sym = require('../lib/symlink');
var rimraf = require('rimraf');

test('setup', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out')) fs.mkdirSync(__dirname + '/fixtures/out');
  assert.end();
});

test('[install] installs a package', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'protozero1.5.1.tar.gz');
  var dst = path.join(__dirname + '/fixtures/out', 'protozero/1.5.1');
  var outfile = path.join(__dirname + '/fixtures/out', 'protozero/1.5.1', 'include', 'protozero', 'byteswap.hpp');
  var url = 'http://fakeurl.com';

  var packageList = [{
    name: 'protozero',
    version: '1.5.1',
    headers: true,
    os: null,
    awsPath: 'headers/protozero/1.5.1.tar.gz',
    src: url,
    dst: dst
  }];

  var buffer = fs.readFileSync(src);

  var mockStream = new stream.PassThrough();
  mockStream.push(buffer);
  mockStream.end();

  sinon.spy(mockStream, 'pipe');
  sinon.spy(log, 'info');

  sinon.stub(needle, 'get').returns(mockStream);

  index.install(packageList, function() {
    sinon.assert.calledOnce(mockStream.pipe);
    sinon.assert.calledTwice(log.info);
    assert.equal(log.info.getCall(0).args[0], 'check');
    assert.equal(log.info.getCall(0).args[1], 'checked for protozero (not found locally)');
    assert.equal(log.info.getCall(1).args[0], 'tarball');
    assert.equal(log.info.getCall(1).args[1], 'done parsing tarball for protozero');
    assert.equal(fs.existsSync(outfile), true);
    fse.removeSync(path.join(__dirname + '/fixtures/out', 'protozero'));
    log.info.restore();
    needle.get.restore();
    assert.end();
  });
});

test('[symlink] links files', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/mason_packages/.link')) fse.mkdirpSync(__dirname + '/fixtures/out/mason_packages/.link');

  var appDir = process.cwd();
  var symlinkPath = path.join(appDir, 'test/fixtures/out/mason_packages/.link');

  var paths = [
    [appDir + '/test/fixtures/headers/protozero/1.5.1',
      symlinkPath
    ],
    [appDir + '/test/fixtures/osx-x86_64/cairo/1.14.8',
      symlinkPath
    ]
  ];

  sinon.spy(log, 'info');

  sinon.stub(sym, 'buildLinkPaths').returns(paths);

  var proto = path.join(appDir + '/test/fixtures/out', 'mason_packages/.link', 'include', 'protozero', 'byteswap.hpp');
  var cairo = path.join(appDir + '/test/fixtures/out', 'mason_packages/.link', 'include', 'cairo', 'cairo-ft.h');
  var masonPath = './test/fixtures/fake-mason-versions.ini';

  index.link(masonPath, function() {
    assert.equal(fs.existsSync(proto), true);
    assert.equal(fs.existsSync(cairo), true);
    assert.equal(log.info.getCall(0).args[0], 'Symlinked: ');
    fse.removeSync(path.join(__dirname, '/fixtures/out', 'mason_packages/.link'));
    sym.buildLinkPaths.restore();
    log.info.restore();
    assert.end();
  });
});

test('cleanup', function(assert) {
  rimraf(__dirname + '/fixtures/out', function(err) {
    assert.ifError(err);
    assert.end();
  });
});

test('[symlink] file to link doesnt exist', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/mason_packages/.link')) fse.mkdirpSync(__dirname + '/fixtures/out/mason_packages/.link');

  var appDir = process.cwd();
  var symlinkPath = path.join(appDir, 'test/fixtures/out/mason_packages/.link');

  var paths = [
    [appDir + '/test/fixtures/headers/protozro/1.5.1',
      symlinkPath
    ],
    [appDir + '/test/fixtures/osx-x86_64/ciro/1.14.8',
      symlinkPath
    ]
  ];

  sinon.spy(log, 'info');

  sinon.stub(sym, 'buildLinkPaths').returns(paths);

  var masonPath = './test/fixtures/fake-mason-versions.ini';

  index.link(masonPath, function(err) {
    assert.equal(/ENOENT: no such file or directory/.test(err.message), true);
    fse.remove(path.join(__dirname, '/fixtures/out', 'mason_packages/.link'), function(err) {
      if (err) return console.error(err);
    });
    sym.buildLinkPaths.restore();
    log.info.restore();
    assert.end();
  });
});

test('cleanup', function(assert) {
  rimraf(__dirname + '/fixtures/out', function(err) {
    assert.ifError(err);
    assert.end();
  });
});
