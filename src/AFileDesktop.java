// test789d3 : USAG Lib basic io Abstract File Desktop

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

public class AFileDesktop implements AFile {
    private final File file;

    public AFileDesktop(String path) {
        this.file = new File(path);
    }
    
    public AFileDesktop(File file) {
        this.file = file;
    }

    @Override
    public InputStream openInput() throws IOException {
        return new FileInputStream(file);
    }

    @Override
    public OutputStream openOutput() throws IOException {
        // Ensure parent exists
        if (file.getParentFile() != null) file.getParentFile().mkdirs();
        return new FileOutputStream(file);
    }

    @Override
    public String getName() { return file.getName(); }

    @Override
    public long getSize() { return file.length(); }
    
    @Override
    public boolean isDirectory() { return file.isDirectory(); }
}