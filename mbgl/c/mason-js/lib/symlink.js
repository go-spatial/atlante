var fs = require('fs');
var fse = require('fs-extra');
var log = require('npmlog');

var filterFunc = function(src, dest) {
  if (src.indexOf('mason.ini') > -1) {
    return false;
  } else {
    var srcStat = fs.lstatSync(src);

    if (srcStat.isSymbolicLink() || srcStat.isFile()) {
      if (fs.existsSync(dest) || srcStat.isSymbolicLink()){
        fse.removeSync(dest);
      }
      fs.symlinkSync(src, dest);
      return false;
    } else {
      return true;
    }
  }
};

function buildLinkPaths(libraries, symlinkPath) {
  var paths = [];

  libraries.forEach(function(l) {
    paths.push([l.dst, symlinkPath]);
  });
  return paths;
}

function symLink(paths, callback) {
  try {
    paths.forEach(function(p) {
      var lsync = fs.lstatSync(p[0]);

      if (lsync.isDirectory() || lsync.isSymbolicLink()){
        fse.copySync(p[0], p[1], {overwrite:true, filter: filterFunc});
        log.info('Symlinked: ', p[0], 'to ', p[1]);
      }

    });
    return callback(null, true);
  } catch (e) {
    return callback(e);
  }
}

module.exports = {
  buildLinkPaths: buildLinkPaths,
  symLink: symLink,
  filterFunc: filterFunc
};
