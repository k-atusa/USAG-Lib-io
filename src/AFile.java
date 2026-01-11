// test789d2 : USAG Lib basic io Abstract File

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

public interface AFile {
    InputStream openInput() throws IOException;
    OutputStream openOutput() throws IOException;
    String getName();
    long getSize();
    boolean isDirectory();
}