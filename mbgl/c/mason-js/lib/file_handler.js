var fs = require('fs');
var path = require('path');
var packagePath;
var os = require('os');
var platform = os.platform();
var MASON_BUCKET = 'https://s3.amazonaws.com/mason-binaries/';
var appDir = process.cwd();
var url = require('url');

function checkOS() {
  switch (platform) {
  case 'darwin':
    return 'osx-x86_64';
  case 'linux':
    return 'linux-x86_64';
  default:
    throw new Error(`${platform} is not a platform option support by mason`);
  }
}

function buildParams(p, type) {
  if (typeof p === 'object' && type === 'compiled') {
    // check OS here
    var osPlatform = checkOS();
    p.os = osPlatform;
    p.headers = false;
    packagePath = path.join(p.os, p.name, p.version + '.tar.gz');
    p.awsPath = packagePath;
    p.src = url.resolve(MASON_BUCKET, p.awsPath).replace(/\s/g, '');
    p.dst = path.join(appDir, 'mason_packages', p.awsPath).replace(/\s/g, '').replace('.tar.gz', '');
  } else if (typeof p === 'object' && type === 'header') {
    packagePath = path.join('headers', p.name, p.version + '.tar.gz');
    p.awsPath = packagePath;
    p.headers = true;
    p.os = '';
    p.src = url.resolve(MASON_BUCKET, p.awsPath).replace(/\s/g, '');
    p.dst = path.join(appDir, 'mason_packages', p.awsPath).replace(/\s/g, '').replace('.tar.gz', '');
  }else{
    p = null;
  }
  if (p) return p;
}

function generatePackageObject(p){
  var singlePackage = p.split('=');

  if (singlePackage.length !== 2){
    throw new Error('Invalid package syntax');
  }

  var packageObject = {
    'name': singlePackage[0],
    'version': singlePackage[1]
  };
  return packageObject;
}

function parseLibraries(fileContents, callback) {
  var libraries = [];
  var packageType;

  if (fileContents.toString().split('\n')[0] !== '[headers]') {
    var err = new Error();
    err.message = 'Headers must be declared before compiled packages.';
    return callback(err);
  }

  fileContents.toString().split('\n').forEach(function(line) {
    var p;

    if (line.indexOf('=') > -1) {
      p = generatePackageObject(line);
    }else if (line === '[compiled]'){
      packageType = 'compiled';
    }else if (line === '[headers]'){
      packageType = 'header';
    }

    var packageObject = buildParams(p, packageType);

    if (packageObject){
      libraries.push(packageObject);
    }
  });
  return callback(null, libraries);
}

function fileReader(path, callback) {
  fs.readFile(path, function(err, fileContents) {
    if (err) return callback(err);
    parseLibraries(fileContents, function(err, libraries) {
      if (err) return callback(err);
      return callback(null, libraries);
    });
  });
}

function fileWriter(src, package, type, callback){
  var result;

  fs.readFile(src, 'utf8', function(err, data) {
    if (err) {
      return callback(err);
    }

    if (data.indexOf(package) > -1){
      return callback(new Error('Package could not be saved, already exists in mason-versions.ini.'));
    }

    if(!data.includes(type)){
      if (type === 'header'){
        data = '[headers]\n' + data;
      }else if (type == 'compiled'){
        data = data + '[compiled]\n';
      }
      result = type === 'header' ? data.replace(/s]/g, 's]' + '\n' + package) : data.replace(/d]/g, 'd]' + '\n' + package);
    }else{
      result = type === 'header' ? data.replace(/s]/g, 's]' + '\n' + package) : data.replace(/d]/g, 'd]' + '\n' + package);
    }


    fs.writeFile(src, result, 'utf8', function (err) {
      if (err) return callback(err);
      return callback(null, true);
    });
  });
}

module.exports = {
  fileReader: fileReader,
  buildParams: buildParams,
  generatePackageObject: generatePackageObject,
  fileWriter:fileWriter
};
