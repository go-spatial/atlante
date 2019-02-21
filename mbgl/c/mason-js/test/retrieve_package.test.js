var test = require('tape');
var fs = require('fs');
var path = require('path');
var sinon = require('sinon');
var needle = require('needle');
var retriever = require('../lib/retrieve_package');
var rimraf = require('rimraf');
var stream = require('stream');
var log = require('npmlog');
var fse = require('fs-extra');

global.appRoot = process.cwd();

test('setup', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out')) fs.mkdirSync(__dirname + '/fixtures/out');
  assert.end();
});

test('[place binary] places binary', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/protozero/1.5.1')) fse.mkdirpSync(__dirname + '/fixtures/out/protozero/1.5.1');

  var src = path.join(__dirname + '/fixtures/', 'protozero1.5.1.tar.gz');
  var dst = path.join(__dirname + '/fixtures/out', 'protozero/1.5.1');
  var outfile = path.join(__dirname + '/fixtures/out', 'protozero/1.5.1', 'include', 'protozero', 'byteswap.hpp');
  var url = 'http://fakeurl.com';

  var options = {
    name: 'protozero',
    version: '1.5.1',
    headers: true,
    os: null,
    awsPath: 'headers/protozero/1.5.1.tar.gz',
    src: url,
    dst: dst
  };

  var buffer = fs.readFileSync(src);

  var mockStream = new stream.PassThrough();
  mockStream.push(buffer);
  mockStream.end();

  sinon.spy(mockStream, 'pipe');
  sinon.spy(log, 'info');

  sinon.stub(needle, 'get').returns(mockStream);

  retriever.placeBinary(options, function() {
    sinon.assert.calledOnce(mockStream.pipe);
    sinon.assert.calledOnce(log.info);
    assert.equal(log.info.getCall(0).args[0], 'tarball');
    assert.equal(log.info.getCall(0).args[1], 'done parsing tarball for protozero');
    assert.equal(fs.existsSync(outfile), true);
    fse.remove(path.join(__dirname + '/fixtures/out', 'protozero'), err => {
      if (err) return console.error(err);
    });
    log.info.restore();
    needle.get.restore();
    assert.end();
  });
});

test('[place binary] gets a needle error', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/protozero1.5.1')) fs.mkdirSync(__dirname + '/fixtures/out/protozero1.5.1');

  var options = {
    name: 'boost',
    version: '1.3.0',
    headers: true,
    os: null,
    awsPath: 'headers/boost/1.3.0.tar.gz',
    src: 'http://fakeurl.com',
    dst: 'dst'
  };

  const mockStream = {};
  mockStream.on = function(event, callback) {
    if (event === 'error') {
      return callback(new Error('there was a needle error'));
    }
  };

  mockStream.pipe = sinon.stub();
  mockStream.pipe.on = sinon.stub();

  sinon.stub(needle, 'get').returns(mockStream);

  retriever.placeBinary(options, function(err) {
    assert.equal(err.message, 'there was a needle error');
    needle.get.restore();
    assert.end();
  });
});

test('[place binary] needle returns status code error ', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/protozero1.5.1')) fs.mkdirSync(__dirname + '/fixtures/out/protozero1.5.1');

  var options = {
    name: 'boost',
    version: '1.3.0',
    headers: true,
    os: null,
    awsPath: 'headers/boost/1.3.0.tar.gz',
    src: 'http://fakeurl.com',
    dst: 'dst'
  };

  const mockStream = {};
  mockStream.on = function(event, callback) {
    if (event === 'response') {
      var res = { statusCode: 400 };
      return callback(res);
    }
  };

  mockStream.pipe = sinon.stub();

  sinon.stub(needle, 'get').returns(mockStream);
  retriever.placeBinary(options, function(err) {
    assert.equal(err.message, '400 status code downloading tarball http://fakeurl.com');
    needle.get.restore();
    assert.end();
  });
});

test('[place binary] needle returns close error ', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out/protozero1.5.1')) fs.mkdirSync(__dirname + '/fixtures/out/protozero1.5.1');

  var options = {
    name: 'boost',
    version: '1.3.0',
    headers: true,
    os: null,
    awsPath: 'headers/boost/1.3.0.tar.gz',
    src: 'http://fakeurl.com',
    dst: 'dst'
  };

  const mockStream = {};
  mockStream.on = function(event, callback) {
    if (event === 'close') {
      return callback(new Error());
    }
  };

  mockStream.pipe = sinon.stub();

  sinon.stub(needle, 'get').returns(mockStream);

  retriever.placeBinary(options, function(err) {
    assert.equal(err.message, 'Connection closed while downloading tarball file');
    needle.get.restore();
    assert.end();
  });
});

test('[check library] logs package already exists', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'boost/1.3.0.tar.gz');
  var dst = path.join(__dirname + '/fixtures/', 'boost/1.3.0');

  var options = {
    name: 'boost',
    version: '1.3.0',
    headers: true,
    os: null,
    awsPath: 'headers/boost/1.3.0.tar.gz',
    src: src,
    dst: dst
  };

  sinon.spy(log, 'info');

  retriever.checkLibraryExists(options, function(err, res) {
    if (err) console.log(err);
    assert.equal(log.info.getCall(0).args[1], 'Success: boost already installed');
    assert.equal(res, true);
    log.info.restore();
    assert.end();
  });
});

test('[check library] creates directory paths', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'boost/1.3.0.tar.gz');
  var dst = path.join(__dirname + '/fixtures/out', 'boost/1.3.0');

  var options = {
    name: 'boost',
    version: '1.3.0',
    headers: true,
    os: null,
    awsPath: 'headers/boost/1.3.0.tar.gz',
    src: src,
    dst: dst
  };

  retriever.checkLibraryExists(options, function(err, res) {
    assert.equal(res, false);
    assert.equal(fs.existsSync(__dirname + '/fixtures/out/boost/1.3.0'), true);
    fse.remove(path.join(__dirname + '/fixtures/out', 'boost'), function(err) {
      if (err) return console.error(err);
    });
    assert.end();
  });
});

test('cleanup', (assert) => {
  rimraf(__dirname + '/fixtures/out', function(err) {
    assert.ifError(err);
    assert.end();
  });
});
