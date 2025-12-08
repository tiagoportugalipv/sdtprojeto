# sudo apt-get install build-essential git cmake libopenblas-dev libgflags-dev -y

projectDir="$PWD"
installDir="$projectDir/local"

cd /tmp

git clone https://github.com/facebookresearch/faiss.git
cd faiss

cmake -B build  \
  -DFAISS_ENABLE_GPU=OFF \
  -DFAISS_ENABLE_PYTHON=OFF \
  -DFAISS_ENABLE_C_API=ON \
  -DBUILD_SHARED_LIBS=ON \
  -DCMAKE_INSTALL_PREFIX=$installDir .

make -C build -j faiss
make -C build install

cd $projectDir
