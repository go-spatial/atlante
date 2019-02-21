var test = require('tape');
var os = require('os');
var path = require('path');
var platform = os.platform();
var reader = require('../lib/file_handler.js');
var exec = require('child_process').exec;
var appDir = process.cwd();
var helpText = 'Usage:\n  mason-js install \n\n  or \n\n  mason-js install <package> <package type>\n\nDescription:\n  mason-js is a JS client for mason that installs c++ packages locally (both header-only and compiled). mason-js can install all packages declared in a mason-versions.ini file or it can install a single package. \n\nExample:\n  mason-js install  \n\n  OR\n\n  mason-js install protozero=1.5.1 --type=header \n\nOptions:\n  --type [header or compiled]\n';
var rimraf = require('rimraf');
var fs = require('fs');

test('setup', function(assert) {
  if (!fs.existsSync(__dirname + '/fixtures/out')) fs.mkdirSync(__dirname + '/fixtures/out');
  assert.end();
});

var headerPackage = {
  name: 'protozero',
  version: '1.5.1',
  headers: true,
  os: '',
  awsPath: 'headers/protozero/1.5.1.tar.gz',
  src: 'https://s3.amazonaws.com/mason-binaries/headers/protozero/1.5.1.tar.gz',
  dst: appDir + '/mason_packages/headers/protozero/1.5.1'
};

var command_path = path.join(__dirname,'../bin/mason-js');

var system;

if (platform === 'darwin') {
  system = 'osx-x86_64';
} else if (platform === 'linux') {
  system = 'linux-x86_64';
}

var compiledPackage = {
  name: 'ccache',
  version: '3.6.4',
  headers: false,
  os: `${system}`,
  awsPath: `${system}/ccache/3.6.4.tar.gz`,
  src: `https://s3.amazonaws.com/mason-binaries/${system}/ccache/3.6.4.tar.gz`,
  dst: appDir + `/mason_packages/${system}/ccache/3.6.4`
};

test('reads ini file correctly', function(assert) {
  var masonPath = './test/fixtures/fake-mason-versions.ini';

  reader.fileReader(masonPath, function(err, result) {
    assert.equal(result.length, 3);
    assert.deepEqual(result[0], headerPackage);
    assert.deepEqual(result[2], compiledPackage);
    assert.end();
  });
});

test('[package object] generates package object correctly', function(assert) {
  var p = 'protozero=1.5.1';
  var expected = {
    name: 'protozero',
    version: '1.5.1'
  };

  var object = reader.generatePackageObject(p);
  assert.deepEqual(object, expected);
  assert.end();
});

test('[package object] invalid package', function(assert) {
  var p = 'protozero1.5.1';

  assert.throws(function(){
    reader.generatePackageObject(p);
  }, /Invalid package syntax/, 'Should throw syntax error');
  assert.end();
});


test('read incorrect ini file', function(assert) {
  var masonPath = './test/fixtures/wrong-order.ini';

  reader.fileReader(masonPath, function(err) {
    assert.equal(err.message, 'Headers must be declared before compiled packages.');
    assert.end();
  });
});

test('ini file does not exist', function(assert) {
  var masonPath = './test/fixtures/no-file.ini';
  var msg = 'ENOENT: no such file or directory, open \'./test/fixtures/no-file.ini\'';
  reader.fileReader(masonPath, function(err) {
    assert.equal(err.message, msg);
    assert.end();
  });
});

test('[mason-js] missing args', function(assert) {
  exec(command_path, function(err, stdout, stderr) {
    assert.ok(err);
    assert.equal(stdout, helpText, 'no stdout');
    assert.equal(stderr, 'ERR! Error: missing mason-js args \n', 'expected args');
    assert.end();
  });
});

test('[mason-js] missing package type', function(assert) {
  exec(command_path + ' install protozero=1.5.1', (err, stdout, stderr) => {
    assert.ok(err);
    assert.equal(stdout, helpText, 'no stdout');
    assert.equal(stderr, 'ERR! Error: include package type with package info: example protozero=1.5.1 --type=header \n', 'expected args');
    assert.end();
  });
});

test('[add package to file] adds header package to mason-versions.ini', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'fake-mason-versions.ini');
  var dst = path.join(__dirname + '/fixtures/out', 'fake-mason-versions.ini');
  var content = fs.readFileSync(src);
  fs.writeFileSync(dst, content);

  var package = 'crazynewpackage=1.5.1';
  var type = 'header';
  var expected = '[headers]\ncrazynewpackage=1.5.1\nprotozero=1.5.1\nsparsepp=0.9.5\n[compiled]\nccache=3.6.4';

  reader.fileWriter(dst,package, type, function(err, res) {
    assert.equal(res, true);
    var data = fs.readFileSync(dst, 'utf8');
    assert.equal(data, expected);
    assert.equal(/crazynewpackage=1.5.1/.test(data), true);
    assert.end();
  });
});

test('[add package to file] adds compiled package to mason-versions.ini', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'fake-mason-versions.ini');
  var dst = path.join(__dirname + '/fixtures/out', 'fake-mason-versions.ini');
  var content = fs.readFileSync(src);
  fs.writeFileSync(dst, content);

  var package = 'crazynewpackage=1.5.1';
  var type = 'compiled';
  var expected = '[headers]\nprotozero=1.5.1\nsparsepp=0.9.5\n[compiled]\ncrazynewpackage=1.5.1\nccache=3.6.4';

  reader.fileWriter(dst,package, type, function(err, res) {
    assert.equal(res, true);
    var data = fs.readFileSync(dst, 'utf8');
    assert.equal(data, expected);
    assert.equal(/crazynewpackage=1.5.1/.test(data), true);
    assert.end();
  });
});

test('[add package to file] adds [compiled] header and package to mason-versions.ini', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'mv-no-compiled.ini');
  var dst = path.join(__dirname + '/fixtures/out', 'mv-no-compiled.ini');
  var content = fs.readFileSync(src);
  fs.writeFileSync(dst, content);

  var package = 'crazynewpackage=1.5.1';
  var type = 'compiled';
  var expected = '[headers]\nboost=1.65.1\nprotozero=1.5.1\n[compiled]\ncrazynewpackage=1.5.1\n';

  reader.fileWriter(dst,package, type, function(err, res) {
    assert.equal(res, true);
    var data = fs.readFileSync(dst, 'utf8');
    assert.equal(data, expected);
    assert.equal(/compiled/.test(data), true);
    assert.equal(/crazynewpackage=1.5.1/.test(data), true);
    assert.end();
  });
});

test('[add package to file] adds [headers] header and header package to mason-versions.ini', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'mv-no-header.ini');
  var dst = path.join(__dirname + '/fixtures/out', 'mv-no-header.ini');
  var content = fs.readFileSync(src);
  fs.writeFileSync(dst, content);

  var package = 'crazynewpackage=1.5.1';
  var type = 'header';
  var expected = '[headers]\ncrazynewpackage=1.5.1\n[compiled]\nllvm=32.3';

  reader.fileWriter(dst,package, type, function(err, res) {
    assert.equal(res, true);
    var data = fs.readFileSync(dst, 'utf8');
    assert.equal(data, expected);
    assert.equal(/compiled/.test(data), true);
    assert.equal(/crazynewpackage=1.5.1/.test(data), true);
    assert.end();
  });
});

test('[add package to file] does not write package already in file', function(assert) {
  var src = path.join(__dirname + '/fixtures/', 'fake-mason-versions.ini');

  var package = 'protozero=1.5.1';
  var type = 'headers';

  reader.fileWriter(src,package, type, function(err) {
    assert.equal(err.message, 'Package could not be saved, already exists in mason-versions.ini.');
    assert.end();
  });
});

test('cleanup', function(assert) {
  rimraf(__dirname + '/fixtures/out', (err) => {
    assert.ifError(err);
    assert.end();
  });
});
