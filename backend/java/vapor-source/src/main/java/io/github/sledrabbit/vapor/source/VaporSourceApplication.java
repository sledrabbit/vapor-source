package io.github.sledrabbit.vapor.source;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@SpringBootApplication
public class VaporSourceApplication {

    @RequestMapping("/")
    String home() {
        return "Hello, world!";
    }

	public static void main(String[] args) {
		SpringApplication.run(VaporSourceApplication.class, args);
	}

}
