#include <stdlib.h>
#include <sys/types.h>
#include <unistd.h>

int main (int argc, char** argv)
{

  int count = 1;
  int sleep_time = 30;

  if (argc > 1) {
    count = atoi(argv[1]);
  }
  if (argc > 2) {
    sleep_time = atoi(argv[2]);
  }

  for (int i = 0; i < count; i++) {
      pid_t child_pid;
      child_pid = fork ();
      if (child_pid == 0) {
        sleep(4);
        exit(0);
      }
  }      

  sleep(sleep_time);
  return 0;
}
