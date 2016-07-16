#include <stdlib.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdio.h>

#    define O_RDWR  0002

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
        /* become a daemon process */

        /* become your own process group leader */
        if (setsid ( ) == -1)
            return 1;
        /* set the working directory to the root directory */
        if (chdir ("/") == -1)
            return 1;

        /* close stdin, stdout, stderr */
        close(0); close(1); close(2);

        /* redirect fd's 0,1,2 to /dev/null */
        open ("/dev/null", O_RDWR);
        /* stdin */
        dup (0);
        /* stdout */
        dup (0);
        /* stderror */

        sleep(sleep_time);
        exit(0);
      }
  }

  return 0;
}
