const path = require('path');

module.exports = {
  mode: 'production',
  // optimization: {
  //   minimize: false
  // },
  entry: {
    index: path.resolve(__dirname, 'dist', 'cjs', 'index.js'),
  },
  output: {
    path: path.resolve(__dirname, 'dist', 'umd'),
    filename: '[name].min.js',
    libraryTarget: 'umd',
    library: 'CAIP',
    umdNamedDefine: true,
    globalObject: 'this',
  },
};
