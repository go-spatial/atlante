var loader = require('./lib/retrieve_package.js');
var sym = require('./lib/symlink.js');
var path = require('path');
var reader = require('./lib/file_handler.js');
var d3 = require('d3-queue');

/* eslint-disable */

function link(masonPath, callback){
// linting is disabled because it errors on callback param

/* eslint-disable */

  reader.fileReader(masonPath, function(err, packages){
    if (err) return callback(err);
    var paths = sym.buildLinkPaths(packages,path.join(process.cwd(), '/mason_packages/.link'));
    sym.symLink(paths, function(err, result){
      if (err) return callback(err);
      return callback(null, result);
    });
  });
}

function install(packageList, callback) {
  var libraries = packageList;
  var q = d3.queue(1);

  libraries.forEach(function(options) {
    if (options) {
      loader.checkLibraryExists(options, function(err, exists) {
        if (err) return callback(err);
        if (!exists) {
          q.defer(loader.placeBinary, options);
        }
      });
    }
  });

  q.awaitAll(function(err, result) {
    if (err) return callback(err);
    return callback(null);
  });
}



module.exports = {
  install:install,
  link:link};
