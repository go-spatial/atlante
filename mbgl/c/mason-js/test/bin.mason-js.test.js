var test = require('tape');
var fs = require('fs');
var path = require('path');
var sinon = require('sinon');
var needle = require('needle');
var reader = require('../lib/file_handler');
var mason = require('../bin/mason-js');
var rimraf = require('rimraf');
var stream = require('stream');
var log = require('npmlog');
var fse = require('fs-extra');
var index = require('../');
var appDir = process.cwd();
var sym = require('../lib/symlink.js');

test('setup', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out')) fs.mkdirSync(__dirname + '/fixtures/out');
  assert.end();
});

test('[install] installs a package from mason-versions.ini', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'protozero1.5.1.tar.gz');
  var dst = path.join(__dirname + '/fixtures/out', 'protozero/1.5.1');
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

  sinon.stub(reader, 'fileReader').callsFake(function(masonPath, callback){
    return callback(null, packageList);
  });

  var buffer = fs.readFileSync(src);

  var mockStream = new stream.PassThrough();
  mockStream.push(buffer);
  mockStream.end();

  sinon.spy(mockStream, 'pipe');
  sinon.spy(log, 'info');

  sinon.stub(needle, 'get').returns(mockStream);

  var masonPath = './test/fixtures/fake-mason-versions.ini';
  var args = { _: [ 'install' ] };

  mason.run(args, masonPath, function(err, result) {
    // sinon.assert.calledOnce(mockStream.pipe);
    assert.equal(log.info.getCall(0).args[0], 'Mason Package Install Starting');
    assert.equal(log.info.getCall(1).args[0], 'check');
    assert.equal(log.info.getCall(1).args[1], 'checked for protozero (not found locally)');
    assert.equal(log.info.getCall(2).args[0], 'tarball');
    assert.equal(log.info.getCall(2).args[1], 'done parsing tarball for protozero');
    assert.equal(result, true);

    fse.removeSync(path.join(__dirname + '/fixtures/out', 'protozero'));
    reader.fileReader.restore();
    log.info.restore();
    needle.get.restore();
    assert.end();
  });
});

test('[install] no mason-versions.ini', function(assert) {

  sinon.stub(reader, 'fileReader').callsFake(function(masonPath, callback){
    var err = new Error('File doesnt exist');
    return callback(err);
  });

  var masonPath = './test/fixtures/bad-verions.ini';
  var args = { _: [ 'install' ] };

  mason.run(args, masonPath, function(err) {
    assert.equal(err.message, 'File doesnt exist');
    reader.fileReader.restore();
    assert.end();
  });
});

test('[installs] single package', function(assert) {

  sinon.stub(index, 'install').callsFake(function(packages, callback){
    return callback(null, true);
  });

  var masonPath = './test/fixtures/fake-mason-versions.ini';
  var args = { _: [ 'install', 'protozero=1.5.1' ], type: 'header' };

  mason.run(args, masonPath, function(err, result) {
    assert.equal(result, true);
    index.install.restore();
    assert.end();
  });
});

test('[links] single package', function(assert) {
  var args = { _: [ 'link', 'protozero=1.5.1' ], type: 'header' };

  var symlinkPath = path.join(global.appRoot, 'test/fixtures/out/mason_packages/.link');
  var masonPath = './test/fixtures/fake-mason-versions.ini';

  var paths = [
    [appDir + '/test/fixtures/headers/protozero/1.5.1',
      symlinkPath
    ]
  ];

  sinon.stub(sym, 'buildLinkPaths').returns(paths);

  var proto = path.join(appDir + '/test/fixtures/out', 'mason_packages/.link', 'include', 'protozero', 'byteswap.hpp');

  sinon.spy(log, 'info');

  mason.run(args, masonPath, function() {
    assert.equal(fs.existsSync(proto), true);
    log.info.restore();
    sym.buildLinkPaths.restore();
    assert.end();
  });
});


test('cleanup', function(assert) {
  rimraf(__dirname + '/fixtures/out', function(err) {
    assert.ifError(err);
    assert.end();
  });
});
