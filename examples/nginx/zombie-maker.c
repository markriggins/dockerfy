#include <stdlib.h>
#include <sys/types.h>
#include <unistd.h>

/**
  Create argv[1] child processes, sleep argv[2] seconds and then exit
  without waiting for their exit codes or catching their signals

  This leaves argv[1] Zombie processes that are normally cleaned
  up by the init daemon, if init is running; however, docker containers
  typically do not have an init process!!   So the zombies can pile up
  and consume the entire process table, and finally crash the system.
*/
int main (int argc, char** argv)
{

  int count = 2;
  int sleep_time = 1;

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
